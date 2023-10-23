package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type User struct {
	ID         string
	SecretCode string
	Name       string
	Email      string
	Playlists  []Playlist
}

type Playlist struct {
	ID    string
	Name  string
	Songs []Song
}

type Song struct {
	ID       string
	Name     string
	Composer string
	URL      string
}

var (
	users     = make(map[string]User)     // Key is SecretCode
	playlists = make(map[string]Playlist) // Key is Playlist ID
	mu        sync.Mutex
	randSrc   = rand.NewSource(time.Now().UnixNano())
)

func generateID() string {
	return fmt.Sprintf("%d", randSrc.Int63())
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	// Assuming secret code comes as a query parameter
	secretCode := r.URL.Query().Get("secretCode")

	if user, exists := users[secretCode]; exists {
		json.NewEncoder(w).Encode(user)
		return
	}

	sendError(w, "Invalid secret code", http.StatusUnauthorized)
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user.ID = generateID()
	user.SecretCode = generateID()
	// Ensure each user has at least one playlist with one song.
	defaultPlaylist := Playlist{
		ID:   generateID(),
		Name: "Default",
		Songs: []Song{
			{
				ID:       generateID(),
				Name:     "Default Song",
				Composer: "Default Composer",
				URL:      "http://default.url",
			},
		},
	}
	user.Playlists = append(user.Playlists, defaultPlaylist)
	users[user.SecretCode] = user

	json.NewEncoder(w).Encode(user)
}

func createPlaylist(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	// Assuming secret code and playlist name come as query parameters
	secretCode := r.URL.Query().Get("secretCode")
	playlistName := r.URL.Query().Get("playlistName")

	if user, exists := users[secretCode]; exists {
		newPlaylist := Playlist{
			ID:   generateID(),
			Name: playlistName,
			Songs: []Song{
				{
					ID:       generateID(),
					Name:     "Default Song",
					Composer: "Default Composer",
					URL:      "http://default.url",
				},
			},
		}

		user.Playlists = append(user.Playlists, newPlaylist)
		users[secretCode] = user

		json.NewEncoder(w).Encode(newPlaylist)
		return
	}

	sendError(w, "Invalid secret code", http.StatusUnauthorized)
}

func deleteSongFromPlaylist(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	// Assuming secret code and song ID come as query parameters
	secretCode := r.URL.Query().Get("secretCode")
	songID := r.URL.Query().Get("songID")
	playlistID := r.URL.Query().Get("playlistID")

	if user, exists := users[secretCode]; exists {
		for i, playlist := range user.Playlists {
			if playlist.ID == playlistID {
				if len(playlist.Songs) == 1 {
					sendError(w, "Playlist cannot be empty", http.StatusBadRequest)
					return
				}

				for j, song := range playlist.Songs {
					if song.ID == songID {
						user.Playlists[i].Songs = append(playlist.Songs[:j], playlist.Songs[j+1:]...)
						users[secretCode] = user
						json.NewEncoder(w).Encode(playlist)
						return
					}
				}
			}
		}
	}

	sendError(w, "Invalid request", http.StatusBadRequest)
}

func viewProfile(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	secretCode := r.URL.Query().Get("secretCode")
	if user, exists := users[secretCode]; exists {
		json.NewEncoder(w).Encode(user)
		return
	}
	sendError(w, "Invalid secret code", http.StatusUnauthorized)
}

func addSongToPlaylist(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	secretCode := r.URL.Query().Get("secretCode")
	playlistID := r.URL.Query().Get("playlistID")

	var newSong Song
	if err := json.NewDecoder(r.Body).Decode(&newSong); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if user, exists := users[secretCode]; exists {
		for i, playlist := range user.Playlists {
			if playlist.ID == playlistID {
				newSong.ID = generateID()
				user.Playlists[i].Songs = append(playlist.Songs, newSong)
				users[secretCode] = user
				json.NewEncoder(w).Encode(newSong)
				return
			}
		}
		sendError(w, "Invalid playlist ID", http.StatusNotFound)
		return
	}
	sendError(w, "Invalid secret code", http.StatusUnauthorized)
}

func deletePlaylist(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	secretCode := r.URL.Query().Get("secretCode")
	playlistID := r.URL.Query().Get("playlistID")

	if user, exists := users[secretCode]; exists {
		for i, playlist := range user.Playlists {
			if playlist.ID == playlistID {
				user.Playlists = append(user.Playlists[:i], user.Playlists[i+1:]...)
				users[secretCode] = user
				json.NewEncoder(w).Encode(user)
				return
			}
		}
		sendError(w, "Invalid playlist ID", http.StatusNotFound)
		return
	}
	sendError(w, "Invalid secret code", http.StatusUnauthorized)
}

func getSongDetail(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	secretCode := r.URL.Query().Get("secretCode")
	songID := r.URL.Query().Get("songID")

	if user, exists := users[secretCode]; exists {
		for _, playlist := range user.Playlists {
			for _, song := range playlist.Songs {
				if song.ID == songID {
					json.NewEncoder(w).Encode(song)
					return
				}
			}
		}
		sendError(w, "Invalid song ID", http.StatusNotFound)
		return
	}
	sendError(w, "Invalid secret code", http.StatusUnauthorized)
}

func sendError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func main() {
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/viewProfile", viewProfile)
	http.HandleFunc("/createPlaylist", createPlaylist)
	http.HandleFunc("/deleteSongFromPlaylist", deleteSongFromPlaylist)
	http.HandleFunc("/addSongToPlaylist", addSongToPlaylist)
	http.HandleFunc("/deletePlaylist", deletePlaylist)
	http.HandleFunc("/getSongDetail", getSongDetail)
	http.ListenAndServe(":8080", nil)
}
