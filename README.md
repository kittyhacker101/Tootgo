# Tootgo
A rewrite of Tootbot (https://github.com/corbindavenport/tootbot) in Golang.

## Compiling
To compile Tootgo, first run `go get -d ./...` to fetch dependencies, and then run `go build -ldflags="-s -w" -tags netgo` to compile the program into a static binary.

## Configuring
To configure Tootgo, open the conf.json file, and replace the placeholder values with the required logins/API keys. It is also heavily recommended to change the posting interval (reddit.interval) from one minute to a higher number, to prevent the bot from being rate limited.
Note that the reddit.subreddit configuration option must not include the /r/ prefix (ex, if you wanted to mirror /r/me_irl, you would set reddit.subreddit to "me_irl").
