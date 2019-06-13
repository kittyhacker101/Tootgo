# Tootgo
A rewrite of Tootbot (https://github.com/corbindavenport/tootbot) in Golang. Designed for managing multiple rapid-posting Twitter bots.

## Compiling
To compile Tootgo, first run `go get -d ./...` to fetch dependencies, and then run `go build -ldflags="-s -w" -tags netgo` to compile the program into a static binary. You must download the [Golang compiler](https://golang.org/dl/) before you can compile the program.

## Configuring
To configure Tootgo, open the conf.json file, and replace the placeholder values with the required logins/API keys. Note that the reddit.subreddits configuration option must not include the /r/ prefix (ex, if you wanted to mirror /r/me_irl, you would set reddit.subreddits to "me_irl").

## Advanced

### timing.allowoffset
The bot has a builtin algorithm to try to show high quality posts sooner than they will appear on other Reddit bots. This is still a work in progress, and can end up making the bot post too frequently. You can enable this by changing timing.allowoffset to true. When this is enabled, the bot will automatically adjust timing.interval, and it will discard some posts which are slowly getting upvotes. You can increase or reduce the minimum amount of upvotes per hour by changing the reddit.minuprate configuration option.

### reddit.ignoreunknown
If the bot is unable to detect an image in a Reddit post, it will simply skip it. If you would like to have the bot post a link to Reddit posts, even if it can't find a valid image in it, then you can do so by changing reddit.ignoreunknown to false.

### reddit.fixbadimages
Often, badly encoded images can make it onto Reddit, and they may result in an image being skipped due to an error, excessive bandwidth usage (especially with GIF files), or (in rare cases) the bot software crashing. This option uses FFMPEG to re-encode images into the WebP format before uploading. This saves bandwidth and fixes corrupt images, at the expensive of CPU time and slightly reduced image quality. This option is experimental, and requires FFMPEG to be installed. You can enable it by changing reddit.fixbadimages to true.