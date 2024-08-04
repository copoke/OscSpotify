package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"github.com/zmb3/spotify"
)

var (
	auth                  spotify.Authenticator
	ch                    = make(chan *spotify.Client)
	state                 = "your-unique-state"
	prevInSliderState     bool
	globalProxValue       float64
	isInSlider            bool
	isInVolume            bool
	prevInVolumeState     bool
	globalVolumeProxValue float64
)

func main() {
	clientID, clientSecret := getCredentials()

	setupSpotifyAuth(clientID, clientSecret)
	setupHttpServer()

	spotifyClient := <-ch
	printCurrentUser(spotifyClient)

	setupOSCServer(spotifyClient)
}

func getCredentials() (clientID, clientSecret string) {
	credentialsFile := "spotify_credentials.txt"

	// Check if credentials file exists
	if _, err := os.Stat(credentialsFile); os.IsNotExist(err) {
		// Prompt for credentials
		fmt.Println("Enter your Spotify Client ID:")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		clientID = scanner.Text()

		fmt.Println("Enter your Spotify Client Secret:")
		scanner.Scan()
		clientSecret = scanner.Text()

		// Save credentials
		file, err := os.Create(credentialsFile)
		if err != nil {
			log.Fatalf("Unable to create credentials file: %v", err)
		}
		defer file.Close()

		_, err = file.WriteString(clientID + "\n" + clientSecret)
		if err != nil {
			log.Fatalf("Unable to write credentials to file: %v", err)
		}
	} else {
		// Read credentials from file
		file, err := os.Open(credentialsFile)
		if err != nil {
			log.Fatalf("Unable to open credentials file: %v", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		scanner.Scan()
		clientID = scanner.Text()
		scanner.Scan()
		clientSecret = scanner.Text()
	}

	return clientID, clientSecret
}

func setupSpotifyAuth(clientID, clientSecret string) {
	redirectURL := "http://localhost:8080/callback"

	scopes := []string{
		spotify.ScopeUserReadCurrentlyPlaying,
		spotify.ScopePlaylistReadPrivate,
		spotify.ScopeUserModifyPlaybackState,
		spotify.ScopeUserReadPlaybackState,
	}

	auth = spotify.NewAuthenticator(redirectURL, scopes...)
	auth.SetAuthInfo(clientID, clientSecret)

	url := auth.AuthURL(state)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)
}

func setupHttpServer() {
	http.HandleFunc("/callback", completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})
	go http.ListenAndServe(":8080", nil)
}

func printCurrentUser(client *spotify.Client) {
	user, err := client.CurrentUser()
	if err != nil {
		log.Fatalf("failed to get current user: %v", err)
	}
	fmt.Println("You are logged in as:", user.ID)
}

func setupOSCServer(spotifyClient *spotify.Client) {
	client := osc.NewClient("127.0.0.1", 9000)
	go trackCurrentPlayingSong(spotifyClient, client)

	addr := "127.0.0.1:9001"
	d := osc.NewStandardDispatcher()
	setupOSCHandlers(d, spotifyClient, client)

	server := &osc.Server{
		Addr:       addr,
		Dispatcher: d,
	}
	server.ListenAndServe()
}

func setupOSCHandlers(dispatcher *osc.StandardDispatcher, spotifyClient *spotify.Client, oscClient *osc.Client) {
	dispatcher.AddMsgHandler("/avatar/parameters/OSC_AUDIO_CONTROLS_PLAY_PAUSE", func(msg *osc.Message) {
		handlePlayPauseSong(spotifyClient, msg)
	})
	dispatcher.AddMsgHandler("/avatar/parameters/OSC_AUDIO_CONTROLS_NEXT", func(msg *osc.Message) {
		handleNextSong(spotifyClient, msg)
	})
	dispatcher.AddMsgHandler("/avatar/parameters/OSC_AUDIO_CONTROLS_PREVIOUS", func(msg *osc.Message) {
		handlePreviousSong(spotifyClient, msg)
	})
	dispatcher.AddMsgHandler("/avatar/parameters/volumeSliderProx", handleVolumeValue)
	dispatcher.AddMsgHandler("/ramp", handleIncomingOSCMessage)
	dispatcher.AddMsgHandler("/avatar/parameters/proxValue", func(msg *osc.Message) {
		handleProxValue(msg)
	})
	dispatcher.AddMsgHandler("/avatar/parameters/shuffleSongState", func(msg *osc.Message) {
		handleShuffleSong(spotifyClient, msg)
	})
	dispatcher.AddMsgHandler("/avatar/parameters/repeatSongState", func(msg *osc.Message) {
		handleRepeatState(spotifyClient, msg)
	})
	dispatcher.AddMsgHandler("/avatar/parameters/inSlider", func(msg *osc.Message) {
		handleInSlider(spotifyClient, oscClient, msg)
	})
	dispatcher.AddMsgHandler("/avatar/parameters/inVolumeSlider", func(msg *osc.Message) {
		handleInVolume(spotifyClient, oscClient, msg)
	})
}

func writeSongNameToFile(songName, artistName string) {
	file, err := os.OpenFile("songConfig.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatalf("failed to open file: %v", err)
	}
	defer file.Close()

	songInfo := fmt.Sprintf("%s - %s\n", songName, artistName)
	_, err = file.WriteString(songInfo)
	if err != nil {
		log.Fatalf("failed to write to file: %v", err)
	}
}

func handleIncomingOSCMessage(msg *osc.Message) {
	if len(msg.Arguments) == 0 {
		log.Println("No arguments in OSC message")
		return
	}

	intVal, ok := msg.Arguments[0].(float32)
	if !ok {
		log.Println("First argument in OSC message is not int32")
		return
	}

	vrcMsg := osc.NewMessage("/avatar/parameters/chan1")
	vrcMsg.Append(intVal)

	client := osc.NewClient("127.0.0.1", 9000)
	if err := client.Send(vrcMsg); err != nil {
		log.Printf("Failed to send VRC message: %v", err)
	}
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s", st, state)
	}

	// use the token to get an authenticated client
	client := auth.NewClient(tok)
	fmt.Fprintf(w, "Login Completed!")
	ch <- &client
}

func roundToDecimal(value float64, decimalPlaces int) float64 {
	shift := math.Pow(10, float64(decimalPlaces))
	return math.Round(value*shift) / shift
}

func getCurrentTrackDuration(client *spotify.Client) (int, error) {
	currentlyPlaying, err := client.PlayerCurrentlyPlaying()
	if err != nil {
		return 0, err
	}
	if currentlyPlaying.Item == nil {
		return 0, fmt.Errorf("no track is currently playing")
	}

	return currentlyPlaying.Item.Duration, nil
}

func trackCurrentPlayingSong(client *spotify.Client, oscClient *osc.Client) {
	for {
		currentlyPlaying, err := client.PlayerCurrentlyPlaying()
		if err != nil {
			log.Printf("Error retrieving currently playing song: %v\n", err)
			time.Sleep(10 * time.Second)
			continue
		}

		if currentlyPlaying.Item == nil {
		} else {
			currentTrack := currentlyPlaying.Item
			songName := currentTrack.Name
			playerState, err := client.PlayerState()
			if err != nil {
				log.Printf("Error retrieving player state: %v\n", err)
				return
			}
			if !isInSlider {
				artistName := currentTrack.Artists[0].Name
				progressTrack := currentlyPlaying
				progressMs := progressTrack.Progress
				durationMs := progressTrack.Item.Duration
				progress := float64(progressMs) / float64(durationMs)
				roundProgress := roundToDecimal(progress, 2)
				progressFloat := float32(roundProgress)
				writeSongNameToFile(songName, artistName)
				message := osc.NewMessage("/avatar/parameters/slider")
				message.Append(progressFloat)
				oscClient.Send(message)
				message2 := osc.NewMessage("/avatar/parameters/isPlaying")
				message2.Append(playerState.Playing)
				oscClient.Send(message2)
			}
		}
		time.Sleep(2 * time.Second)
	}
}

func handleShuffleSong(client *spotify.Client, msg *osc.Message) {
	shuffleStateInt, err := parseVRCInt(msg)
	if err != nil {
		fmt.Println(err)
		return
	}

	var shuffleState bool
	if shuffleStateInt == 1 {
		shuffleState = true
	} else {
		shuffleState = false
	}

	err = client.Shuffle(shuffleState)
	if err != nil {
		fmt.Println("Failed to set shuffle state:", err)
	} else {
		fmt.Printf("Spotify shuffle state set to %v\n", shuffleState)
	}
}

func handleRepeatState(client *spotify.Client, msg *osc.Message) {
	log.Println(msg)

	type RepeatState string

	const (
		RepeatOff     RepeatState = "off"
		RepeatTrack   RepeatState = "track"
		RepeatContext RepeatState = "context"
	)

	repeatStateInt, err := parseVRCInt(msg)
	if err != nil {
		fmt.Println(err)
		return
	}

	var repeatState RepeatState
	switch repeatStateInt {
	case 0:
		repeatState = RepeatOff
	case 1:
		repeatState = RepeatContext
	case 2:
		repeatState = RepeatTrack
	default:
		fmt.Printf("Invalid repeat state: %d\n", repeatStateInt)
		return
	}

	// Call Spotify API to set repeat state
	err = client.Repeat(string(repeatState)) // Convert to string
	if err != nil {
		fmt.Println("Failed to set repeat state:", err)
	} else {
		fmt.Printf("Spotify repeat state set to %v\n", repeatState)
	}
}

func handleInSlider(client *spotify.Client, oscClient *osc.Client, msg *osc.Message) {
	inSlider, err := parseVRCBool(msg)
	isInSlider = inSlider
	if err != nil {
		fmt.Println(err)
		return
	}

	if prevInSliderState && !inSlider {
		duration, err := getCurrentTrackDuration(client)
		if err != nil {
			log.Printf("Error retrieving current track duration: %v\n", err)
			return
		}
		percentage := -globalProxValue + 1
		timestamp := int(float32(duration) * float32(percentage))
		message := osc.NewMessage("/avatar/parameters/slider")
		message.Append(roundToDecimal(percentage, 2))
		oscClient.Send(message)
		log.Println("sending:", message)
		err = client.Seek(timestamp)
	}
	if err != nil {
		log.Printf("Error seeking to timestamp: %v\n", err)
	}
	prevInSliderState = inSlider
}

func handleInVolume(client *spotify.Client, oscClient *osc.Client, msg *osc.Message) {
	inVolume, err := parseVRCBool(msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	isInVolume = inVolume
	if prevInVolumeState && !inVolume {
		log.Println(globalVolumeProxValue)
		percentage := 100 * globalVolumeProxValue
		client.Volume(int(percentage))
		log.Println("successfully set spotify volume to:", percentage, "%")
	}

	prevInVolumeState = inVolume
}

func handleProxValue(msg *osc.Message) {
	if isInSlider {
		proxValue, err := parseVRCFloat(msg)
		if err != nil {
			fmt.Println(err)
			return
		}
		globalProxValue = float64(proxValue)
	}
}

func handleVolumeValue(msg *osc.Message) {
	if isInVolume {
		volumeValue, err := parseVRCFloat(msg)
		if err != nil {
			fmt.Println(err)
			return
		}
		globalVolumeProxValue = float64(volumeValue)
	}
}

func parseVRCBool(msg *osc.Message) (bool, error) {
	message := strings.Trim(msg.String(), " ")
	fmt.Println(message)
	if strings.HasSuffix(message, ",T true") {
		return true, nil
	} else if strings.HasSuffix(message, ",F false") {
		return false, nil
	}

	return false, fmt.Errorf("Unexpected value: %v\n", message)
}

func parseVRCFloat(msg *osc.Message) (float64, error) {
	message := strings.Trim(msg.String(), " ")
	messageSlice := strings.Split(message, " ")
	floatStr := messageSlice[len(messageSlice)-1]
	return strconv.ParseFloat(floatStr, 64)
}

func parseVRCInt(msg *osc.Message) (int, error) {
	if len(msg.Arguments) == 0 {
		return 0, fmt.Errorf("no arguments in OSC message")
	}
	if value, ok := msg.Arguments[0].(int32); ok {
		return int(value), nil
	}

	return 0, fmt.Errorf("first argument in OSC message is not an int32")
}

func handlePlayPauseSong(spotifyClient *spotify.Client, msg *osc.Message) {
	wasSelected, err := parseVRCBool(msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	if !wasSelected {
		return
	}

	// Determine whether to play or pause based on current playback state
	playerState, err := spotifyClient.PlayerState()
	if err != nil {
		fmt.Printf("Failed to get current player state: %v\n", err)
		return
	}

	if playerState.Playing {
		err = spotifyClient.Pause()
		if err != nil {
			fmt.Printf("Failed to pause playback: %v\n", err)
		} else {
			fmt.Println("Playback paused")
		}
	} else {
		err = spotifyClient.Play()
		if err != nil {
			fmt.Printf("Failed to resume playback: %v\n", err)
		} else {
			fmt.Println("Playback resumed")
		}
	}
}

func handleNextSong(spotifyClient *spotify.Client, msg *osc.Message) {
	wasSelected, err := parseVRCBool(msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	if !wasSelected {
		return
	}

	err = spotifyClient.Next()
	if err != nil {
		fmt.Printf("Failed to skip to the next song: %v\n", err)
	}
}

func handlePreviousSong(spotifyClient *spotify.Client, msg *osc.Message) {
	wasSelected, err := parseVRCBool(msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	if !wasSelected {
		return
	}

	err = spotifyClient.Previous()
	if err != nil {
		fmt.Printf("Failed to skip to the previous song: %v\n", err)
	}
}
