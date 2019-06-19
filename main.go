package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/dghubble/oauth1"
	"github.com/jzelinskie/geddit"
	"github.com/lkramer/go-twitter/twitter"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Conf struct {
	Bots []struct {
		Twitter struct {
			Token string `json:"token"`
			ToknS string `json:"tokensecret"`
			Conk  string `json:"key"`
			Cons  string `json:"keysecret"`
		} `json:"twitter"`
		Timing struct {
			Time float64 `json:"interval"`
			Adj  bool    `json:"allowoffset"`
		} `json:"timing"`
		Reddit struct {
			Mon  []string `json:"subreddits"`
			Minu int      `json:"minuprate"`
			Igv  bool     `json:"ignoreunknown"`
		} `json:"reddit"`
	} `json:"bots"`
	Reddit struct {
		User string `json:"username"`
		Pass string `json:"password"`
		Fix  bool   `json:"fixbadimages"`
	} `json:"reddit"`
}

var (
	conf Conf
)

func download(id int, session *geddit.LoginSession, status *twitter.StatusService, media *twitter.MediaService) {
	var posts []string
	var subreddit string
	for _, sub := range conf.Bots[id].Reddit.Mon {
		subreddit = subreddit + "+" + sub
	}
	subreddit = strings.TrimPrefix(subreddit, "+")

	submissions, err := session.SubredditSubmissions(subreddit, geddit.HotSubmissions, geddit.ListingOptions{
		Limit: 60,
	})
	if err != nil {
		fmt.Println("Unable to get subreddit posts!")
		return
	}

	f, err := os.OpenFile(strconv.Itoa(id)+"_posts.txt", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Unable to load database!")
		return
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		posts = append(posts, scanner.Text())
	}
	sort.Strings(posts)

	for d, s := range submissions {
		i := sort.SearchStrings(posts, s.ID)
		if i < len(posts) && posts[i] == s.ID {
			continue
		}
		upv := float64(s.Ups) / time.Since(time.Unix(int64(s.DateCreated), 0)).Hours()
		fmt.Println("Subreddits: " + subreddit + " | Upvotes/hour: " + strconv.Itoa(int(upv)) + " | Post timing: " + strconv.FormatFloat(conf.Bots[id].Timing.Time, 'f', -1, 64))
		if conf.Bots[id].Timing.Adj {
			switch {
			case d <= 15:
				conf.Bots[id].Timing.Time = conf.Bots[id].Timing.Time - 0.8
			case d <= 20:
				conf.Bots[id].Timing.Time = conf.Bots[id].Timing.Time - 0.4
			case d <= 25:
				conf.Bots[id].Timing.Time = conf.Bots[id].Timing.Time - 0.2
			case d >= 55:
				conf.Bots[id].Timing.Time = conf.Bots[id].Timing.Time + 0.4
			case d >= 50:
				conf.Bots[id].Timing.Time = conf.Bots[id].Timing.Time + 0.2
			case d >= 45:
				conf.Bots[id].Timing.Time = conf.Bots[id].Timing.Time + 0.1
			case d > 40:
				conf.Bots[id].Timing.Time = conf.Bots[id].Timing.Time + 0.05
			case int(upv) < conf.Bots[id].Reddit.Minu:
				fmt.Println("Post https://redd.it/" + s.ID + " has been skipped!")
				continue
			}
		}
		f.WriteString(s.ID + "\n")
		f.Close()

		link := "none"
		fmt.Println(s.Title + " https://redd.it/" + s.ID + " (Link : " + s.URL + ")")
		if strings.HasSuffix(s.URL, ".png") || strings.HasSuffix(s.URL, ".jpg") || strings.HasSuffix(s.URL, ".gif") || strings.HasSuffix(s.URL, ".webp") {
			link = s.URL
		}
		if strings.HasSuffix(s.URL, ".gifv") {
			link = strings.TrimSuffix(s.URL, ".gifv") + ".gif"
		}
		if strings.HasPrefix(s.URL, "http://imgur.com/") {
			link = "https://i.imgur.com/" + strings.TrimPrefix(s.URL, "http://imgur.com/") + ".jpg"
		}
		if strings.HasPrefix(s.URL, "https://imgur.com/") {
			link = "https://i.imgur.com/" + strings.TrimPrefix(s.URL, "https://imgur.com/") + ".jpg"
		}
		if link == "none" && !conf.Bots[id].Reddit.Igv {
			fmt.Println("Linking https://redd.it/" + s.ID + "... | Post depth : " + strconv.Itoa(d+1))
			_, _, err = status.Update(s.Title+" https://redd.it/"+s.ID, &twitter.StatusUpdateParams{
				PossiblySensitive: &s.IsNSFW,
			})
			if err != nil {
				fmt.Println("Unable to post tweet!")
				return
			}
			fmt.Println("Link posted to Twitter!")
			return
		} else if link == "none" {
			fmt.Println("Unable to find image, skipping...")
			return
		}

		isGif := false
		if strings.HasSuffix(s.URL, ".gif") {
			isGif = true
		}

		fmt.Println("Downloading " + link + "... | Post depth : " + strconv.Itoa(d+1))
		client := &http.Client{
			Timeout: 10 * time.Second,
		}
		resp, err := client.Get(link)
		if err != nil || resp.StatusCode >= 400 {
			fmt.Println("Unable to download image!")
			resp.Body.Close()
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Println("Unable to read response!")
			return
		}

		// I'll name it a .png even if it isn't, and let Twitter sort it out ¯\_(ツ)_/¯
		f, err = os.Create(os.TempDir() + "/" + s.ID + ".png")
		if err != nil {
			fmt.Println("Unable to write response!")
			return
		}
		f.Write(body)

		fi, err := f.Stat()
		f.Close()
		if err != nil {
			fmt.Println("Unable to get image info!")
			return
		}

		if fi.Size() > 4950000 {
			fmt.Println("File is too big to upload!")
			return
		}

		var img *twitter.Media
		if conf.Reddit.Fix {
			if isGif {
				cmnd := exec.Command("ffmpeg", "-i", os.TempDir()+"/"+s.ID+".png", "-y", os.TempDir()+"/"+s.ID+".gif")
				fmt.Println("Processing image...")
				err := cmnd.Run()
				os.Remove(os.TempDir() + "/" + s.ID + ".png")
				if err != nil {
					fmt.Println("Failed to process image! Is ffmpeg installed?")
					os.Remove(os.TempDir() + "/" + s.ID + ".gif")
					return // Don't attempt to upload the unprocessed image. There's a chance that FFMPEG is installed, and the image is too corrupt for it to process.
				}

				fi, err := os.Stat(os.TempDir() + "/" + s.ID + ".gif")
				f.Close()
				if err != nil {
					fmt.Println("Unable to get image info!")
					return
				}

				if fi.Size() > 4950000 {
					fmt.Println("File is too big to upload!")
					return
				}

				img, _, err = media.UploadFile(os.TempDir() + "/" + s.ID + ".gif")
				os.Remove(os.TempDir() + "/" + s.ID + ".gif")
				if err != nil {
					fmt.Println("Unable to upload media!")
					return
				}
			} else {
				cmnd := exec.Command("ffmpeg", "-i", os.TempDir()+"/"+s.ID+".png", "-lossless", "0", "-compression_level", "6", "-q:v", "80", "-y", os.TempDir()+"/"+s.ID+".webp")
				fmt.Println("Processing image...")
				err := cmnd.Run()
				os.Remove(os.TempDir() + "/" + s.ID + ".png")
				if err != nil {
					fmt.Println("Failed to process image! Is ffmpeg installed?")
					os.Remove(os.TempDir() + "/" + s.ID + ".webp")
					return // Don't attempt to upload the unprocessed image. There's a chance that FFMPEG is installed, and the image is too corrupt for it to process.
				}

				fi, err := os.Stat(os.TempDir() + "/" + s.ID + ".webp")
				f.Close()
				if err != nil {
					fmt.Println("Unable to get image info!")
					return
				}

				if fi.Size() > 4950000 {
					fmt.Println("File is too big to upload!")
					return
				}

				img, _, err = media.UploadFile(os.TempDir() + "/" + s.ID + ".webp")
				os.Remove(os.TempDir() + "/" + s.ID + ".webp")
				if err != nil {
					fmt.Println("Unable to upload media!")
					return
				}
			}
		} else {
			img, _, err = media.UploadFile(os.TempDir() + "/" + s.ID + ".png")
			os.Remove(os.TempDir() + "/" + s.ID + ".png")
			if err != nil {
				fmt.Println("Unable to upload media!")
				return
			}
		}

		_, _, err = status.Update(s.Title+" https://redd.it/"+s.ID, &twitter.StatusUpdateParams{
			MediaIds:          []int64{img.MediaID},
			PossiblySensitive: &s.IsNSFW,
		})
		if err != nil {
			fmt.Println("Unable to post tweet!")
		}
		fmt.Println("Image posted to Twitter!")

		return
	}
	f.Close()
}

func loginAndPost(id int) {
	session, err := geddit.NewLoginSession(
		conf.Reddit.User,
		conf.Reddit.Pass,
		"Tootgo",
	)
	if err != nil {
		fmt.Println("Unable to login to Reddit!")
		return
	}

	config := oauth1.NewConfig(conf.Bots[id].Twitter.Conk, conf.Bots[id].Twitter.Cons)
	token := oauth1.NewToken(conf.Bots[id].Twitter.Token, conf.Bots[id].Twitter.ToknS)
	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)
	download(id, session, client.Statuses, client.Media)
}

func main() {
	data, err := ioutil.ReadFile("conf.json")
	if err != nil {
		fmt.Println("Unable to read config file!")
		return
	}
	if json.Unmarshal(data, &conf) != nil {
		fmt.Println("Unable to parse config file!")
		return
	}

	if len(conf.Bots) == 0 {
		fmt.Println("You must specify at least one bot account for Tootgo to manage.")
	}

	debug.SetGCPercent(20)
	if len(conf.Bots)/16 < runtime.NumCPU() || len(conf.Bots) > 16 {
		runtime.GOMAXPROCS(len(conf.Bots) / 16)
	} else if len(conf.Bots) <= 16 {
		runtime.GOMAXPROCS(1)
	}

	for id := range conf.Bots {
		go func(id int) {
			for {
				loginAndPost(id)
				runtime.GC()
				if conf.Bots[id].Timing.Adj && conf.Bots[id].Timing.Time > 5 {
					time.Sleep(time.Minute * time.Duration(conf.Bots[id].Timing.Time))
				} else {
					time.Sleep(5 * time.Minute)
				}
			}
		}(id)
	}
	cr := make(chan os.Signal, 1)
	signal.Notify(cr, syscall.SIGHUP)
	<-cr
	fmt.Println("Stopping Tootgo...")
}
