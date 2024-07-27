package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/raitonoberu/ytmusic"
	"github.com/wader/goutubedl"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

var (
	_    = godotenv.Load()
	auth = spotifyauth.New(
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(spotifyauth.ScopeUserReadCurrentlyPlaying),
		spotifyauth.WithClientID(os.Getenv("SPOTIFY_CLIENT_ID")),
		spotifyauth.WithClientSecret(os.Getenv("SPOTIFY_CLIENT_SECRET")),
	)
	ch    = make(chan *spotify.Client)
	state = "randomstr"
	redirectURI = os.Getenv("REDIRECT_URI")
	user_username = os.Getenv("TELEGRAM_USER_USERNAME")
	is_spotify_authed = false
)

func main() {
	var client *spotify.Client
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_API_TOKEN"))
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = false
	log.Printf("Telegram bot authorized on account %s", bot.Self.UserName)

	http.HandleFunc("/callback", completeAuth)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	for update := range updates {
		if update.Message != nil {
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
			if update.Message.From.UserName != user_username {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "You are not allowed to use this bot")
				msg.ReplyToMessageID = update.Message.MessageID
				bot.Send(msg)
			} else {
				if update.Message.Text == "/np" {
					if is_spotify_authed{
						found_track, err := spotify_to_yt(client)
						var message_text string
						if err != nil {
							message_text = err.Error()
						} else {
							vid_id := found_track.VideoID
							vid_name := found_track.Artists[0].Name + " - " + found_track.Title
							link := "https://music.youtube.com/watch?v=" + vid_id
							path, err := youtube_download(link, vid_name)
							if err != nil {
								message_text = err.Error()
							} else {
								file, err := os.Open(path)
								if err != nil {
									log.Println(err)
									message_text = "ERROR\n\nfile was downloaded, but failed to open it\n\ntry again"
									msg := tgbotapi.NewMessage(update.Message.Chat.ID, message_text)
									msg.ReplyToMessageID = update.Message.MessageID
									bot.Send(msg)
								} else {
									inputFile := tgbotapi.FileReader{Name: vid_name+".mp3", Reader: file}
									footer := "\n\nvia @spotify_np_golang_bot"
									audio := tgbotapi.NewAudio(update.Message.Chat.ID, inputFile)
									audio.Caption = footer
									audio.Duration = int(found_track.Duration)
									audio.Performer = found_track.Artists[0].Name
									audio.Title = found_track.Title
									audio.ReplyToMessageID = update.Message.MessageID
									_, err = bot.Send(audio)
									if err != nil {
										log.Fatal(err)
									} else {
										os.Remove(path)
										log.Println("Sent video to " + update.Message.From.UserName)
									}
									
								}
							}
						}
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, message_text)
						msg.ReplyToMessageID = update.Message.MessageID
						bot.Send(msg)
					} else {
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Please log in to spotify first with /login")
						msg.ReplyToMessageID = update.Message.MessageID
						bot.Send(msg)
					}
				} else if update.Message.Text == "/login" {
					go func() {
						url := auth.AuthURL(state)
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Please log in to Spotify by visiting the following page in your browser: " + url)
						msg.ReplyToMessageID = update.Message.MessageID
						bot.Send(msg)
						client = <-ch
						user, err := client.CurrentUser(context.Background())
						if err != nil {
							log.Fatal(err)
						}
						log.Println("Logged in as:", user.ID)
					}()
					go http.ListenAndServe(":8080", nil)
					
				} else {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Unknown command")
					msg.ReplyToMessageID = update.Message.MessageID
					bot.Send(msg)
				}

			}
		}
	}

}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(r.Context(), state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}
	// use the token to get an authenticated client
	client := spotify.New(auth.Client(r.Context(), tok))
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "Login Completed!")
	is_spotify_authed = true
	ch <- client
}

func yt_music_search(artist_query, track_query string) (vid_id *ytmusic.TrackItem, err error) {
	track_query = strings.ToLower(track_query)
	artist_query = strings.ToLower(artist_query)
	searchQuery := fmt.Sprintf("%s - %s", artist_query, track_query)
	trackSearch := &ytmusic.SearchClient{
						Query:        searchQuery,
						SearchFilter: "EgWKAQIIAWoMEA4QChADEAQQCRAF",
						Language:     "ru",
						Region:       "RU",
					}
	log.Println(searchQuery)

	for i := 0; i < 1000; i++ {
		result, err := trackSearch.Next()
		if err != nil {
			return nil, errors.New("nothing found")
		}

		for _, track := range result.Tracks {
			if strings.ToLower(track.Title) == track_query && strings.ToLower(track.Artists[0].Name) == artist_query {
				return track, nil
			}
		}
	}
	return nil, errors.New("nothing found")
}

func spotify_to_yt(client *spotify.Client) (*ytmusic.TrackItem, error) {
	currentTrack, err := client.PlayerCurrentlyPlaying(context.Background())
	if err != nil {
		err = errors.New("nothing playing")
		return nil, err
	}
	artists := currentTrack.Item.Artists[0].Name
	track := currentTrack.Item.Name
	found_track, err := yt_music_search(artists, track)
	if err != nil {
		err = errors.New("nothing found at yt music")
		return nil, err
	} else {
		return found_track, nil
	}
}

func youtube_download(link, vid_name string) (filePath string, err error) {

	result, err := goutubedl.New(context.Background(), link, goutubedl.Options{DownloadAudioOnly: true})
	if err != nil {
		return
	} 
	
	downloadResult, err := result.Download(context.Background(), "m4a")
	if err != nil {
		return
	}
	defer downloadResult.Close()
	filePath = "vids/" + vid_name + ".mp3"
	f, err := os.Create(filePath)
	if err != nil {
		log.Println(err)
		err = errors.New("failed to create file")
		return
	}
	defer f.Close()
	io.Copy(f, downloadResult)
	return
}