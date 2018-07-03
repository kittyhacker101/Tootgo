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
	"sort"
	"strconv"
	"strings"
	"time"
)

type Conf struct {
	Reddit struct {
		User string `json:"username"`
		Pass string `json:"password"`
		Mon  string `json:"subreddit"`
		Time float64 `json:"interval"`
		Adj bool `json:"allowoffset"`
	} `json:"reddit"`
	Twitter struct {
		Token string `json:"token"`
		ToknS string `json:"tokensecret"`
		Conk  string `json:"key"`
		Cons  string `json:"keysecret"`
		Minu  int `json:"minuprate"`
	} `json:"twitter"`
}

var (
	conf Conf
)

func download(session *geddit.LoginSession, status *twitter.StatusService, media *twitter.MediaService) {
	var posts []string
	submissions, err := session.SubredditSubmissions(conf.Reddit.Mon, geddit.HotSubmissions, geddit.ListingOptions{
		Limit: 50,
	})
	if err != nil {
		fmt.Println("Unable to get subreddit posts!")
		return
	}

	f, err := os.OpenFile("posts.txt", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
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
		if conf.Reddit.Adj {
			upv := s.Ups / int(time.Since(time.Unix(int64(s.DateCreated), 0)).Hours())
			fmt.Println("Upvotes/hour : " + strconv.Itoa(upv) + " | Post timing : " + strconv.FormatFloat(conf.Reddit.Time, 'f', -1, 64))
			switch {
			case d <= 30:
				conf.Reddit.Time = conf.Reddit.Time - 0.1
			case d < 35:
				conf.Reddit.Time = conf.Reddit.Time - 0.05
			case d > 40:
				conf.Reddit.Time = conf.Reddit.Time + 0.05
			case d >= 45:
				conf.Reddit.Time = conf.Reddit.Time + 0.1
			case upv < conf.Twitter.Minu:
				fmt.Println("Post https://redd.it/" + s.ID + " has been skipped!")
				continue
			}
		}
		f.WriteString(s.ID + "\n")
		f.Close()

		link := "none"
		fmt.Println(s.Title + " https://redd.it/" + s.ID + " (Link : " + s.URL + ")")
		if strings.HasSuffix(s.URL, ".png") || strings.HasSuffix(s.URL, ".jpg") || strings.HasSuffix(s.URL, ".gif") {
			link = s.URL
		}
		if strings.HasPrefix(s.URL, "http://imgur.com/") {
			link = "https://i.imgur.com/" + strings.TrimPrefix(s.URL, "http://imgur.com/") + ".jpg"
		}
		if strings.HasPrefix(s.URL, "https://imgur.com/") {
			link = "https://i.imgur.com/" + strings.TrimPrefix(s.URL, "https://imgur.com/") + ".jpg"
		}
		if link == "none" {
			fmt.Println("Linking https://redd.it/" + s.ID + "... | Post depth : " + strconv.Itoa(d+1))
			_, _, err = status.Update(s.Title+" https://redd.it/"+s.ID, &twitter.StatusUpdateParams{
				PossiblySensitive: &s.IsNSFW,
			})
			if err != nil {
				fmt.Println("Unable to post tweet!")
			}
			fmt.Println("Link posted to Twitter!")
			return
		}

		fmt.Println("Downloading " + link + "... | Post depth : " + strconv.Itoa(d+1))
		resp, err := http.Get(link)
		if err != nil {
			fmt.Println("Unable to download image!")
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Unable to read response!")
			return
		}
		resp.Body.Close()

		// I'll name it a .png even if it isn't, and let Twitter sort it out ¯\_(ツ)_/¯
		f, err = os.Create(os.TempDir() + "/" + s.ID + ".png")
		if err != nil {
			fmt.Println("Unable to write response!")
			return
		}
		f.Write(body)
		f.Close()

		img, _, err := media.UploadFile(os.TempDir() + "/" + s.ID + ".png")
		if err != nil {
			fmt.Println("Unable to upload media!")
			return
		}
		os.Remove(os.TempDir() + "/" + s.ID + ".png")

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

	session, err := geddit.NewLoginSession(
		conf.Reddit.User,
		conf.Reddit.Pass,
		"Tootgo",
	)
	if err != nil {
		fmt.Println("Unable to login to Reddit!")
		return
	}

	config := oauth1.NewConfig(conf.Twitter.Conk, conf.Twitter.Cons)
	token := oauth1.NewToken(conf.Twitter.Token, conf.Twitter.ToknS)
	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)

	download(session, client.Statuses, client.Media)

	ticker := time.NewTicker(time.Minute * time.Duration(conf.Reddit.Time))
	for {
		select {
		case <-ticker.C:
			download(session, client.Statuses, client.Media)
			ticker.Stop()
			ticker = time.NewTicker(time.Minute * time.Duration(conf.Reddit.Time))
		}
	}
}
