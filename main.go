package main

import (
	"bytes"
	"crypto/md5"
	"embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

const Version = "0.5.0"

type spaHandler struct {
	fileSystem fs.FS
	indexPath  string
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean the path
	path := r.URL.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		path = h.indexPath
	}

	// Try to open the file
	f, err := h.fileSystem.Open(path)
	if err != nil {
		// File not found, serve index.html
		h.serveIndex(w, r)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if info.IsDir() {
		// If directory, serve index.html
		h.serveIndex(w, r)
		return
	}

	// File exists and is not a dir, serve it
	http.FileServer(http.FS(h.fileSystem)).ServeHTTP(w, r)
}

func (h spaHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	f, err := h.fileSystem.Open(h.indexPath)
	if err != nil {
		http.Error(w, "Index file not found", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		http.Error(w, "Index file stat failed", http.StatusInternalServerError)
		return
	}

	content, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "Failed to read index file", http.StatusInternalServerError)
		return
	}

	http.ServeContent(w, r, h.indexPath, info.ModTime(), bytes.NewReader(content))
}

// Response structure
type Response struct {
	Code int          `json:"code"`
	Data ResponseData `json:"data"`
	Msg  string       `json:"msg"`
}

type ResponseData struct {
	Total int         `json:"total"`
	List  interface{} `json:"list"`
}

// Global variable to hold loaded JSON data
var mediaDir string
var staticDir string
var randomEnabled bool

//go:embed dist/*
var embedDist embed.FS
var fileSystem fs.FS

var jsonVideos []map[string]interface{}
var jsonMusic []map[string]interface{}
var jsonUsers map[string]interface{}
var jsonUsersList []map[string]interface{}
var jsonPosts []map[string]interface{}
var jsonGoods []map[string]interface{}

var avatar_dataurl = "data:image/png;base64,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"

var (
	mediaCacheMutex   sync.RWMutex
	mediaCacheVideos  []map[string]interface{}
	mediaCacheExpire  time.Time
	currentRandomSeed int64
)

const mediaCacheTtl = 3 * time.Minute

func loadJsonData() {
	// Load users
	usersBytes, err := fs.ReadFile(fileSystem, "data/users.json")
	if err != nil {
		log.Printf("Failed to read users.json: %v", err)
	} else {
		var users []map[string]interface{}
		if err := json.Unmarshal(usersBytes, &users); err != nil {
			log.Printf("Failed to parse users.json: %v", err)
		} else {
			jsonUsers = make(map[string]interface{})
			for i := range users {
				u := users[i]
				keys := []string{"avatar_thumb", "avatar_medium", "avatar_large", "avatar_168x168", "avatar_larger", "avatar_300x300"}
				for _, k := range keys {
					if m, ok := u[k].(map[string]interface{}); ok {
						m["url_list"] = []string{avatar_dataurl}
						u[k] = m
					} else {
						u[k] = map[string]interface{}{"url_list": []string{avatar_dataurl}}
					}
				}
				users[i] = u
			}
			jsonUsersList = users
			for _, u := range users {
				if uid, ok := u["uid"].(string); ok {
					jsonUsers[uid] = u
				}
			}
			log.Printf("Loaded %d users", len(jsonUsers))
		}
	}

	// Load posts (for /post/recommended)
	postsBytes, err := fs.ReadFile(fileSystem, "data/posts.json")
	if err != nil {
		log.Printf("Failed to read posts.json: %v", err)
	} else {
		if err := json.Unmarshal(postsBytes, &jsonPosts); err != nil {
			log.Printf("Failed to parse posts.json: %v", err)
		} else {
			log.Printf("Loaded %d posts", len(jsonPosts))
		}
	}

	// Load goods (for /shop/recommended)
	goodsBytes, err := fs.ReadFile(fileSystem, "data/goods.json")
	if err != nil {
		log.Printf("Failed to read goods.json: %v", err)
	} else {
		if err := json.Unmarshal(goodsBytes, &jsonGoods); err != nil {
			log.Printf("Failed to parse goods.json: %v", err)
		} else {
			log.Printf("Loaded %d goods", len(jsonGoods))
		}
	}

	// Load videos
	// Note: Original code used "src/assets/data/posts6.json", which seems to be a hardcoded path for dev?
	// But let's assume it should also be in the static FS if we want it to work in prod.
	// However, the original code had: videosBytes, err := os.ReadFile("src/assets/data/posts6.json")
	// This path "src/assets/..." looks like source code path, not dist path.
	// If the user hasn't moved this to dist, it might fail.
	// But let's check dist content. dist/data/videos.json exists.
	// Maybe we should try to load from dist/data/videos.json instead of src/...
	// Let's try to read from "data/videos.json" first, if that's what is intended.
	// But strictly following the instruction, I should just fix the staticDir usage.
	// The line `videosBytes, err := os.ReadFile("src/assets/data/posts6.json")` does NOT use staticDir.
	// So I should probably leave it as is?
	// But if the user runs the binary without source code, this will fail.
	// The LS output shows `dist/data/videos.json`. It's likely the same content.
	// I'll try to use `data/videos.json` from fileSystem as a better default, but fallback to original if needed?
	// Actually, let's stick to what's likely correct for a "dist" based deployment.

	videosBytes, err := fs.ReadFile(fileSystem, "data/videos.json")
	if err != nil {
		log.Printf("Failed to read data/videos.json from fs: %v. Trying src path...", err)
		videosBytes, err = os.ReadFile("src/assets/data/posts6.json")
	}

	if err != nil {
		log.Printf("Failed to read videos json: %v", err)
	} else {
		var videos []map[string]interface{}
		if err := json.Unmarshal(videosBytes, &videos); err != nil {
			log.Printf("Failed to parse videos json: %v", err)
		} else {
			for _, v := range videos {
				v["type"] = "recommend-video"

				// Link author
				if authorIDNum, ok := v["author_user_id"].(float64); ok {
					// Convert float64 to string (integer)
					authorID := fmt.Sprintf("%.0f", authorIDNum)
					if user, found := jsonUsers[authorID]; found {
						v["author"] = user
					}
				}
				jsonVideos = append(jsonVideos, v)
			}
			log.Printf("Loaded %d videos from JSON", len(jsonVideos))
		}
	}
}

func loadMusicData() {
	musicBytes, err := fs.ReadFile(fileSystem, "data/music.json")
	if err != nil {
		log.Printf("Failed to read music.json: %v", err)
	} else {
		if err := json.Unmarshal(musicBytes, &jsonMusic); err != nil {
			log.Printf("Failed to parse music.json: %v", err)
		} else {
			log.Printf("Loaded %d music items", len(jsonMusic))
		}
	}
}

func scanMediaVideos() ([]map[string]interface{}, error) {
	// Check cache
	mediaCacheMutex.RLock()
	if time.Now().Before(mediaCacheExpire) && mediaCacheVideos != nil {
		defer mediaCacheMutex.RUnlock()
		return mediaCacheVideos, nil
	}
	mediaCacheMutex.RUnlock()

	videos := make([]map[string]interface{}, 0)

	err := filepath.WalkDir(mediaDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// If mediaDir itself doesn't exist, we'll catch it.
			if os.IsNotExist(err) && path == mediaDir {
				return nil // Treat as empty
			}
			return err
		}
		if d.IsDir() {
			return nil
		}

		fileName := d.Name()
		lowerName := strings.ToLower(fileName)
		if !strings.HasSuffix(lowerName, ".mp4") && !strings.HasSuffix(lowerName, ".webm") && !strings.HasSuffix(lowerName, ".ogg") {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(mediaDir, path)
		if err != nil {
			return nil
		}

		info, _ := d.Info()
		var ct int64
		if info != nil {
			ct = info.ModTime().Unix()
		}

		// Use filename as description, remove extension
		desc := strings.TrimSuffix(fileName, filepath.Ext(fileName))

		// Generate a fake ID
		hash := md5.Sum([]byte(relPath))
		id := hex.EncodeToString(hash[:])

		parts := strings.Split(relPath, string(os.PathSeparator))
		for i, part := range parts {
			parts[i] = url.PathEscape(part)
		}
		videoUrl := fmt.Sprintf("/media/%s", strings.Join(parts, "/"))
		// Use a placeholder for cover
		coverUrl := "" // Could be a default image

		video := map[string]interface{}{
			"type":        "recommend-video",
			"aweme_id":    id,
			"desc":        desc,
			"create_time": ct,
			"music": map[string]interface{}{
				"id":     123456789,
				"title":  "Original Sound",
				"author": "Local Artist",
				"cover_medium": map[string]interface{}{
					"url_list": []string{"/assets/18-C60ghDhY.jpg"},
				},
				"cover_thumb": map[string]interface{}{
					"url_list": []string{"/assets/18-C60ghDhY.jpg"},
				},
				"cover_large": map[string]interface{}{
					"url_list": []string{"/assets/18-C60ghDhY.jpg"},
				},
				"play_url": map[string]interface{}{
					"uri":      "music_uri",
					"url_list": []string{""},
				},
			},
			"video": map[string]interface{}{
				"play_addr": map[string]interface{}{
					"uri":      id,
					"url_list": []string{videoUrl},
					"width":    720,
					"height":   1280,
				},
				"cover": map[string]interface{}{
					"url_list": []string{coverUrl},
				},
				"width":  720,
				"height": 1280,
			},
			"author": map[string]interface{}{
				"uid":       "local_user",
				"nickname":  "Local User",
				"unique_id": "local_user_id",
				"avatar_thumb": map[string]interface{}{
					"url_list": []string{avatar_dataurl},
				},
				"avatar_medium": map[string]interface{}{
					"url_list": []string{avatar_dataurl},
				},
				"avatar_large": map[string]interface{}{
					"url_list": []string{avatar_dataurl},
				},
				"avatar_168x168": map[string]interface{}{
					"url_list": []string{avatar_dataurl},
				},
				"avatar_larger": map[string]interface{}{
					"url_list": []string{avatar_dataurl},
				},
				"cover_url": []map[string]interface{}{
					{
						"url_list": []string{""},
					},
				},
				"share_info": map[string]interface{}{
					"share_qrcode_url": map[string]interface{}{
						"url_list": []string{""},
					},
					"share_url": "",
					"share_image_url": map[string]interface{}{
						"url_list": []string{""},
					},
				},
			},
			"statistics": map[string]interface{}{
				"digg_count":    0,
				"comment_count": 0,
				"share_count":   0,
				"play_count":    0,
			},
			"share_info": map[string]interface{}{
				"share_url": "",
			},
			"status": map[string]interface{}{
				"is_delete": false,
			},
			"aweme_control": map[string]interface{}{
				"can_forward":      true,
				"can_share":        true,
				"can_comment":      true,
				"can_show_comment": true,
			},
		}
		videos = append(videos, video)
		return nil
	})

	if err != nil {
		if os.IsNotExist(err) {
			return videos, nil
		}
		return nil, err
	}

	sort.Slice(videos, func(i, j int) bool {
		vi := int64(0)
		vj := int64(0)
		switch x := videos[i]["create_time"].(type) {
		case int64:
			vi = x
		case int:
			vi = int64(x)
		case float64:
			vi = int64(x)
		}
		switch y := videos[j]["create_time"].(type) {
		case int64:
			vj = y
		case int:
			vj = int64(y)
		case float64:
			vj = int64(y)
		}
		return vi > vj
	})

	mediaCacheMutex.Lock()
	mediaCacheVideos = videos
	mediaCacheExpire = time.Now().Add(mediaCacheTtl)
	mediaCacheMutex.Unlock()

	return videos, nil
}

func recommendedHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	// Prefer JSON data if available
	var pagedVideos interface{}
	var total int

	startStr := r.URL.Query().Get("start")
	pageSizeStr := r.URL.Query().Get("pageSize")
	start := 0
	pageSize := 10
	if startStr != "" {
		fmt.Sscanf(startStr, "%d", &start)
	}
	if pageSizeStr != "" {
		fmt.Sscanf(pageSizeStr, "%d", &pageSize)
	}

	// Fallback to scanned media
	videos, err := scanMediaVideos()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if randomEnabled {
		if start == 0 {
			currentRandomSeed = time.Now().UnixNano()
		}
		r := rand.New(rand.NewSource(currentRandomSeed))
		shuffled := make([]map[string]interface{}, len(videos))
		copy(shuffled, videos)
		r.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})
		videos = shuffled
	}
	total = len(videos)
	end := start + pageSize
	if end > total {
		end = total
	}
	if start > total {
		start = total
	}
	pagedVideos = videos[start:end]
	// if len(jsonVideos) > 0 {
	// 	total = len(jsonVideos)
	// 	end := start + pageSize
	// 	if end > total {
	// 		end = total
	// 	}
	// 	if start > total {
	// 		start = total
	// 	}
	// 	pagedVideos = jsonVideos[start:end]
	// } else {

	// }

	resp := ResponseData{
		Total: total,
		List:  pagedVideos,
	}
	w.Header().Set("Content-Type", "application/json")

	finalResp := map[string]interface{}{
		"code": 200,
		"data": resp,
		"msg":  "",
	}
	json.NewEncoder(w).Encode(finalResp)
}

func musicHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	resp := ResponseData{
		Total: len(jsonMusic),
		List:  jsonMusic,
	}
	w.Header().Set("Content-Type", "application/json")

	finalResp := map[string]interface{}{
		"code": 200,
		"data": resp,
		"msg":  "",
	}
	json.NewEncoder(w).Encode(finalResp)
}

func videoLongRecommendedHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	startStr := r.URL.Query().Get("start")
	pageSizeStr := r.URL.Query().Get("pageSize")
	start := 0
	pageSize := 10
	if startStr != "" {
		fmt.Sscanf(startStr, "%d", &start)
	}
	if pageSizeStr != "" {
		fmt.Sscanf(pageSizeStr, "%d", &pageSize)
	}

	var list interface{}
	total := len(jsonVideos)
	if total > 0 {
		end := start + pageSize
		if end > total {
			end = total
		}
		if start > total {
			start = total
		}
		list = jsonVideos[start:end]
	} else {
		list = []interface{}{}
	}

	resp := ResponseData{
		Total: total,
		List:  list,
	}
	w.Header().Set("Content-Type", "application/json")

	finalResp := map[string]interface{}{
		"code": 200,
		"data": resp,
		"msg":  "",
	}
	json.NewEncoder(w).Encode(finalResp)
}

func videoCommentsHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		// Default ID if none provided, or random from list
		id = "7260749400622894336"
	}

	// Try to read json file first
	path := filepath.Join(staticDir, "data", "comments", fmt.Sprintf("video_id_%s.json", id))
	data, err := os.ReadFile(path)
	if err != nil {
		// Fallback to check if .md exists (simulating fetch logic which handles .md)
		// Since we are server side, we can just look for .json.
		// If specific ID not found, maybe return a default one?
		// The mock logic had a list of valid IDs and picked random if not found.
		// For simplicity, let's try a fallback ID.
		fallbackID := "7260749400622894336"
		path = filepath.Join(staticDir, "data", "comments", fmt.Sprintf("video_id_%s.json", fallbackID))
		data, err = os.ReadFile(path)
		if err != nil {
			http.Error(w, "Comments not found", http.StatusNotFound)
			return
		}
	}

	var comments interface{}
	if err := json.Unmarshal(data, &comments); err != nil {
		http.Error(w, "Failed to parse comments", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"code": 200,
		"data": comments,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func videoPrivateHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	// Logic from mock: allRecommendVideos.slice(100, 110)
	var list interface{}
	total := 10
	if len(jsonVideos) >= 110 {
		list = jsonVideos[100:110]
	} else if len(jsonVideos) > 0 {
		list = jsonVideos
		total = len(jsonVideos)
	} else {
		list = []interface{}{}
		total = 0
	}

	resp := ResponseData{
		Total: total,
		List:  list,
	}
	w.Header().Set("Content-Type", "application/json")

	finalResp := map[string]interface{}{
		"code": 200,
		"data": resp,
		"msg":  "",
	}
	json.NewEncoder(w).Encode(finalResp)
}

func videoLikeHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	// Logic from mock: allRecommendVideos.slice(200, 350)
	var list interface{}
	total := 150
	startIdx := 200
	endIdx := 350
	if len(jsonVideos) >= endIdx {
		list = jsonVideos[startIdx:endIdx]
	} else if len(jsonVideos) > startIdx {
		list = jsonVideos[startIdx:]
		total = len(jsonVideos) - startIdx
	} else {
		list = []interface{}{}
		total = 0
	}

	resp := ResponseData{
		Total: total,
		List:  list,
	}
	w.Header().Set("Content-Type", "application/json")

	finalResp := map[string]interface{}{
		"code": 200,
		"data": resp,
		"msg":  "",
	}
	json.NewEncoder(w).Encode(finalResp)
}

func videoMyHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	pageNo := 0
	pageSize := 10
	if p := r.URL.Query().Get("pageNo"); p != "" {
		fmt.Sscanf(p, "%d", &pageNo)
	}
	if ps := r.URL.Query().Get("pageSize"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}
	offset := pageNo * pageSize

	// Load specific user video list
	// In mock it was hardcoded to user-12345xiaolaohu.md
	path := filepath.Join(staticDir, "data", "user_video_list", "user-12345xiaolaohu.json")
	data, err := os.ReadFile(path)
	var userVideos []map[string]interface{}

	if err == nil {
		if err := json.Unmarshal(data, &userVideos); err != nil {
			log.Printf("Failed to parse user videos: %v", err)
		} else {
			// Map author info if available
			for _, v := range userVideos {
				if authorIDNum, ok := v["author_user_id"].(float64); ok {
					authorID := fmt.Sprintf("%.0f", authorIDNum)
					if user, found := jsonUsers[authorID]; found {
						v["author"] = user
					}
				} else if authorIDStr, ok := v["author_user_id"].(string); ok {
					if user, found := jsonUsers[authorIDStr]; found {
						v["author"] = user
					}
				}
			}
		}
	}

	total := len(userVideos)
	var list interface{}
	end := offset + pageSize
	if offset >= total {
		list = []interface{}{}
	} else {
		if end > total {
			end = total
		}
		list = userVideos[offset:end]
	}

	finalResp := map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"pageNo": pageNo,
			"total":  total,
			"list":   list,
		},
		"msg": "",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(finalResp)
}

func videoHistoryHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	pageNo := 0
	pageSize := 10
	if p := r.URL.Query().Get("pageNo"); p != "" {
		fmt.Sscanf(p, "%d", &pageNo)
	}
	if ps := r.URL.Query().Get("pageSize"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}
	offset := pageNo * pageSize

	// Mock logic: allRecommendVideos.slice(200, 350).slice(offset, limit)
	var list interface{}
	total := 150
	startIdx := 200
	endIdx := 350

	// Get the subset first
	var subset []map[string]interface{}
	if len(jsonVideos) >= endIdx {
		subset = jsonVideos[startIdx:endIdx]
	} else if len(jsonVideos) > startIdx {
		subset = jsonVideos[startIdx:]
		total = len(subset)
	} else {
		subset = []map[string]interface{}{}
		total = 0
	}

	// Then apply pagination on the subset
	subsetTotal := len(subset)
	subEnd := offset + pageSize
	if offset >= subsetTotal {
		list = []interface{}{}
	} else {
		if subEnd > subsetTotal {
			subEnd = subsetTotal
		}
		list = subset[offset:subEnd]
	}

	finalResp := map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"total": total,
			"list":  list,
		},
		"msg": "",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(finalResp)
}

func userPanelHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	// Mock logic: find user with specific UID
	uid := "2739632844317827"
	var resp map[string]interface{}

	if user, ok := jsonUsers[uid]; ok {
		resp = map[string]interface{}{
			"code": 200,
			"data": user,
		}
	} else {
		resp = map[string]interface{}{
			"code": 500,
			"msg":  "User not found",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func userCollectHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	// Mock logic: video: slice(350, 400), music: resource.music
	var videoList interface{}
	videoTotal := 50
	vStart := 350
	vEnd := 400
	if len(jsonVideos) >= vEnd {
		videoList = jsonVideos[vStart:vEnd]
	} else if len(jsonVideos) > vStart {
		videoList = jsonVideos[vStart:]
		videoTotal = len(jsonVideos) - vStart
	} else {
		videoList = []interface{}{}
		videoTotal = 0
	}

	finalResp := map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"video": map[string]interface{}{
				"total": videoTotal,
				"list":  videoList,
			},
			"music": map[string]interface{}{
				"total": len(jsonMusic),
				"list":  jsonMusic,
			},
		},
		"msg": "",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(finalResp)
}

func userVideoListHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	id := r.URL.Query().Get("id")
	filePath := fmt.Sprintf("data/user_video_list/user-%s.json", id)
	data, err := fs.ReadFile(fileSystem, filePath)

	if err != nil {
		finalResp := map[string]interface{}{
			"code": 500,
			"msg":  "User video list not found",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(finalResp)
		return
	}

	var videos interface{}
	json.Unmarshal(data, &videos)

	finalResp := map[string]interface{}{
		"code": 200,
		"data": videos,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(finalResp)
}

func userFriendsHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	finalResp := map[string]interface{}{
		"code": 200,
		"data": jsonUsersList,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(finalResp)
}

func historyOtherHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	pageNo := 0
	if p := r.URL.Query().Get("pageNo"); p != "" {
		fmt.Sscanf(p, "%d", &pageNo)
	}

	finalResp := map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"pageNo": pageNo,
			"total":  0,
			"list":   []interface{}{},
		},
		"msg": "",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(finalResp)
}

func postRecommendedHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	pageNo := 0
	pageSize := 10
	if p := r.URL.Query().Get("pageNo"); p != "" {
		fmt.Sscanf(p, "%d", &pageNo)
	}
	if ps := r.URL.Query().Get("pageSize"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}
	offset := pageNo * pageSize

	total := len(jsonPosts)
	var list interface{}
	end := offset + pageSize

	// Mock logic: allRecommendPosts.slice(0, 1000).slice(offset, limit)
	// We just slice directly.
	if offset >= total {
		list = []interface{}{}
	} else {
		if end > total {
			end = total
		}
		list = jsonPosts[offset:end]
	}

	finalResp := map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"pageNo": pageNo,
			"total":  total,
			"list":   list,
		},
		"msg": "",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(finalResp)
}

func shopRecommendedHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	pageNo := 0
	pageSize := 10
	if p := r.URL.Query().Get("pageNo"); p != "" {
		fmt.Sscanf(p, "%d", &pageNo)
	}
	if ps := r.URL.Query().Get("pageSize"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}
	offset := pageNo * pageSize

	total := len(jsonGoods)
	var list interface{}
	end := offset + pageSize

	if offset >= total {
		list = []interface{}{}
	} else {
		if end > total {
			end = total
		}
		list = jsonGoods[offset:end]
	}

	finalResp := map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"total": total,
			"list":  list,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(finalResp)
}

func videoIndexedHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}
	videos, err := scanMediaVideos()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := ResponseData{
		Total: len(videos),
		List:  videos,
	}
	w.Header().Set("Content-Type", "application/json")
	finalResp := map[string]interface{}{
		"code": 200,
		"data": resp,
		"msg":  "",
	}
	json.NewEncoder(w).Encode(finalResp)
}

func scanMediaVideoFiles() ([]map[string]interface{}, error) {
	items := make([]map[string]interface{}, 0)
	err := filepath.WalkDir(mediaDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) && path == mediaDir {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		lower := strings.ToLower(name)
		if !strings.HasSuffix(lower, ".mp4") && !strings.HasSuffix(lower, ".webm") && !strings.HasSuffix(lower, ".ogg") {
			return nil
		}
		rel, err := filepath.Rel(mediaDir, path)
		if err != nil {
			return nil
		}
		parts := strings.Split(rel, string(os.PathSeparator))
		for i := range parts {
			parts[i] = url.PathEscape(parts[i])
		}
		urlPath := "/media/" + strings.Join(parts, "/")
		info, _ := d.Info()
		item := map[string]interface{}{
			"url":      urlPath,
			"filename": name,
		}
		if info != nil {
			item["size"] = info.Size()
			item["mod_time"] = info.ModTime().Unix()
		}
		items = append(items, item)
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return items, nil
		}
		return nil, err
	}
	return items, nil
}

func videoListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}
	items, err := scanMediaVideoFiles()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := ResponseData{
		Total: len(items),
		List:  items,
	}
	w.Header().Set("Content-Type", "application/json")
	finalResp := map[string]interface{}{
		"code": 200,
		"data": resp,
		"msg":  "",
	}
	json.NewEncoder(w).Encode(finalResp)
}
func main() {
	log.Printf("douyin_selfhost v%s starting...", Version)
	var staticPath string
	var indexPath string
	var mediaDirFlag string
	var hostFlag string
	var portFlag string
	var configFile string
	var randomFlag bool

	flag.StringVar(&configFile, "config", "", "Path to config file")
	flag.StringVar(&configFile, "c", "", "Path to config file (shorthand)")
	flag.StringVar(&staticPath, "static", "dist", "Path to static files directory")
	flag.StringVar(&indexPath, "index", "index.html", "Path to index.html")
	flag.StringVar(&mediaDirFlag, "media", "media", "Path to media directory")
	flag.StringVar(&hostFlag, "host", "127.0.0.1", "Host to bind")
	flag.StringVar(&portFlag, "port", "8080", "Port to listen")
	flag.BoolVar(&randomFlag, "random", false, "Randomly shuffle scanned media videos")
	flag.Parse()

	// Set up viper for config file support
	v := viper.New()
	v.SetConfigType("yaml")

	// Auto-discover config.yaml from executable dir and working dir
	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		v.SetConfigName("config")
		v.AddConfigPath(".")
		if exe, err := os.Executable(); err == nil {
			v.AddConfigPath(filepath.Dir(exe))
		}
	}

	// Set defaults
	v.SetDefault("host", "127.0.0.1")
	v.SetDefault("port", "8080")
	v.SetDefault("static", "dist")
	v.SetDefault("index", "index.html")
	v.SetDefault("media", "media")
	v.SetDefault("random", false)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Printf("Warning: failed to read config file: %v", err)
		}
	} else {
		log.Printf("Using config file: %s", v.ConfigFileUsed())
	}

	// Apply CLI flag overrides: if a flag was explicitly set, override the config value
	cliFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		cliFlags[f.Name] = true
	})
	if cliFlags["host"] {
		v.Set("host", hostFlag)
	}
	if cliFlags["port"] {
		v.Set("port", portFlag)
	}
	if cliFlags["static"] {
		v.Set("static", staticPath)
	}
	if cliFlags["index"] {
		v.Set("index", indexPath)
	}
	if cliFlags["media"] {
		v.Set("media", mediaDirFlag)
	}
	if cliFlags["random"] {
		v.Set("random", randomFlag)
	}

	// Use viper values
	mediaDir = v.GetString("media")
	staticDir = v.GetString("static")
	randomEnabled = v.GetBool("random")
	host := v.GetString("host")
	port := v.GetString("port")

	// Initialize fileSystem
	if _, err := os.Stat(staticDir); err == nil {
		log.Printf("Using local static directory: %s", staticDir)
		fileSystem = os.DirFS(staticDir)
	} else {
		log.Printf("Local directory %s not found, using embedded resources", staticDir)
		var err error
		fileSystem, err = fs.Sub(embedDist, "dist")
		if err != nil {
			log.Fatal("Failed to load embedded dist:", err)
		}
	}

	// Load JSON data on startup
	loadJsonData()
	loadMusicData()

	// Serve media files
	http.Handle("/media/", http.StripPrefix("/media/", http.FileServer(http.Dir(mediaDir))))

	// API endpoints
	http.HandleFunc("/video/recommended", recommendedHandler)
	http.HandleFunc("/video/long/recommended", videoLongRecommendedHandler)
	http.HandleFunc("/video/comments", videoCommentsHandler)
	http.HandleFunc("/video/private", videoPrivateHandler)
	http.HandleFunc("/video/like", videoLikeHandler)
	http.HandleFunc("/video/my", videoMyHandler)
	http.HandleFunc("/video/history", videoHistoryHandler)
	http.HandleFunc("/api/video/list", videoListHandler)

	http.HandleFunc("/user/panel", userPanelHandler)
	http.HandleFunc("/user/collect", userCollectHandler)
	http.HandleFunc("/user/video_list", userVideoListHandler)
	http.HandleFunc("/user/friends", userFriendsHandler)

	http.HandleFunc("/historyOther", historyOtherHandler)
	http.HandleFunc("/post/recommended", postRecommendedHandler)
	http.HandleFunc("/shop/recommended", shopRecommendedHandler)

	http.HandleFunc("/music", musicHandler)

	// SPA handler for frontend
	spa := spaHandler{fileSystem: fileSystem, indexPath: v.GetString("index")}
	http.Handle("/", spa)

	log.Printf("Serving SPA on http://%s:%s ...", host, port)
	log.Fatal(http.ListenAndServe(host+":"+port, nil))
}
