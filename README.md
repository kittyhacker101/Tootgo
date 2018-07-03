# Tootgo
A rewrite of Tootbot (https://github.com/corbindavenport/tootbot) in Golang.

## Compiling
To compile Tootgo, first run `go get -d ./...` to fetch dependencies, and then run `go build -ldflags="-s -w" -tags netgo` to compile the program into a static binary. You must download the [Golang compiler](https://dl.google.com/go/go1.10.3.darwin-amd64.pkg) before you can compile the program.

## Configuring
To configure Tootgo, open the conf.json file, and replace the placeholder values with the required logins/API keys. It is also heavily recommended to change the posting interval (reddit.interval) from one minute to a higher number, to prevent the bot from being rate limited.
Note that the reddit.subreddit configuration option must not include the /r/ prefix (ex, if you wanted to mirror /r/me_irl, you would set reddit.subreddit to "me_irl").

## Advanced
The bot has a builtin algorithm to try to show high quality posts sooner than they will appear on other Reddit bots. This is experimental, and it is not recommended to enable this. You can enable this by setting reddit.allowoffset to true. When this is enabled, the bot will automatically adjust reddit.interval, and it will discard some posts which are slowly getting upvotes. You can increase or reduce the minimum amount of upvotes per hour by changing the twitter.minuprate configuration option.
