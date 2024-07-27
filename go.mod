module ch4og/spotify-share-telegram

go 1.22.5

require (
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/joho/godotenv v1.5.1
	github.com/raitonoberu/ytmusic v0.0.0-20240324143733-0e5780514b1d
	github.com/wader/goutubedl v0.0.0-20240725172441-4a4a53c7458b
	github.com/zmb3/spotify/v2 v2.4.2
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	golang.org/x/net v0.23.0 // indirect
	golang.org/x/oauth2 v0.0.0-20210810183815-faf39c7919d5 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

replace github.com/wader/goutubedl => github.com/ch4og/goutubedl v0.0.0-20240727121030-027858ad5af2
