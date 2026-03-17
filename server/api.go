package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

// APIErrorResponse as standard response error
type APIErrorResponse struct {
	ID         string `json:"id"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
}

type ThreadTranslationFailure struct {
	PostID  string `json:"post_id"`
	Message string `json:"message"`
}

type ThreadTranslationResponse struct {
	Translations []*TranslatedMessage       `json:"translations"`
	Failures     []ThreadTranslationFailure `json:"failures"`
}

func (p *Plugin) translatePost(post *model.Post, source, target string) (*TranslatedMessage, error) {
	output, translateErr := p.translateWithScaleway(post.Message, source, target)
	if translateErr != nil {
		return nil, translateErr
	}

	return &TranslatedMessage{
		ID:             post.Id + source + target + strconv.FormatInt(post.UpdateAt, 10),
		PostID:         post.Id,
		SourceLanguage: source,
		SourceText:     post.Message,
		TargetLanguage: target,
		TranslatedText: output.TranslatedText,
		UpdateAt:       post.UpdateAt,
	}, nil
}

func writeAPIError(w http.ResponseWriter, err *APIErrorResponse) {
	b, _ := json.Marshal(err)
	w.WriteHeader(err.StatusCode)
	w.Write(b)
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	if err := p.IsValid(); err != nil {
		http.Error(w, "This plugin is not configured.", http.StatusNotImplemented)
	}

	w.Header().Set("Content-Type", "application/json")

	switch path := r.URL.Path; path {
	case "/api/go":
		p.getGo(w, r)
	case "/api/thread":
		p.getThread(w, r)
	case "/api/get_info":
		p.getInfo(w, r)
	case "/api/set_info":
		p.setInfo(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (p *Plugin) getGo(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized to translate post", http.StatusUnauthorized)
		return
	}

	postID := r.URL.Query().Get("post_id")
	if len(postID) != 26 {
		http.Error(w, "Invalid parameter: post_id", http.StatusBadRequest)
		return
	}

	source := r.URL.Query().Get("source")
	if len(source) < 2 || len(source) > 5 {
		http.Error(w, "Invalid parameter: source", http.StatusBadRequest)
		return
	}
	if languageCodes[source] == "" {
		http.Error(w, "Unsupported parameter: source", http.StatusBadRequest)
		return
	}

	target := r.URL.Query().Get("target")
	if len(target) < 2 || len(target) > 5 {
		http.Error(w, "Invalid parameter: target", http.StatusBadRequest)
		return
	}
	if target == autoLanguage || languageCodes[target] == "" {
		http.Error(w, "Unsupported parameter: target", http.StatusBadRequest)
		return
	}

	post, err := p.API.GetPost(postID)
	if err != nil {
		http.Error(w, "No post to translate", http.StatusBadRequest)
		return
	}

	translated, translateErr := p.translatePost(post, source, target)
	if translateErr != nil {
		http.Error(w, translateErr.Error(), http.StatusBadRequest)
		return
	}

	resp, _ := json.Marshal(translated)
	w.Write(resp)
}

func (p *Plugin) getThread(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized to translate thread", http.StatusUnauthorized)
		return
	}

	postID := r.URL.Query().Get("post_id")
	if len(postID) != 26 {
		http.Error(w, "Invalid parameter: post_id", http.StatusBadRequest)
		return
	}

	source := r.URL.Query().Get("source")
	if len(source) < 2 || len(source) > 5 {
		http.Error(w, "Invalid parameter: source", http.StatusBadRequest)
		return
	}
	if languageCodes[source] == "" {
		http.Error(w, "Unsupported parameter: source", http.StatusBadRequest)
		return
	}

	target := r.URL.Query().Get("target")
	if len(target) < 2 || len(target) > 5 {
		http.Error(w, "Invalid parameter: target", http.StatusBadRequest)
		return
	}
	if target == autoLanguage || languageCodes[target] == "" {
		http.Error(w, "Unsupported parameter: target", http.StatusBadRequest)
		return
	}

	postList, err := p.API.GetPostThread(postID)
	if err != nil || postList == nil {
		http.Error(w, "No thread to translate", http.StatusBadRequest)
		return
	}

	requestedPostIDs := map[string]bool{}
	postIDsParam := r.URL.Query().Get("post_ids")
	if postIDsParam != "" {
		for _, requestedPostID := range strings.Split(postIDsParam, ",") {
			if len(requestedPostID) == 26 {
				requestedPostIDs[requestedPostID] = true
			}
		}
	}

	response := ThreadTranslationResponse{
		Translations: make([]*TranslatedMessage, 0, len(postList.Order)),
		Failures:     make([]ThreadTranslationFailure, 0),
	}
	for _, threadPostID := range postList.Order {
		post := postList.Posts[threadPostID]
		if post == nil || post.Type != "" || post.Message == "" {
			continue
		}
		if len(requestedPostIDs) > 0 && !requestedPostIDs[threadPostID] {
			continue
		}

		translated, translateErr := p.translatePost(post, source, target)
		if translateErr != nil {
			response.Failures = append(response.Failures, ThreadTranslationFailure{
				PostID:  threadPostID,
				Message: translateErr.Error(),
			})
			continue
		}

		response.Translations = append(response.Translations, translated)
	}

	resp, _ := json.Marshal(response)
	w.Write(resp)
}

func (p *Plugin) getInfo(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		// silently return as user is probably not logged in
		return
	}

	info, err := p.getUserInfo(userID)
	if err != nil {
		// silently return as user may not have activated the autotranslation
		return
	}

	resp, _ := json.Marshal(info)
	w.Write(resp)
}

func (p *Plugin) setInfo(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized to set info", http.StatusUnauthorized)
		return
	}

	var info *UserInfo
	json.NewDecoder(r.Body).Decode(&info)
	if info == nil {
		http.Error(w, "Invalid parameter: info", http.StatusBadRequest)
		return
	}

	if err := info.IsValid(); err != nil {
		http.Error(w, fmt.Sprintf("Invalid info: %s", err.Error()), http.StatusBadRequest)
		return
	}

	if info.UserID != userID {
		http.Error(w, "Invalid parameter: user mismatch", http.StatusBadRequest)
		return
	}

	err := p.setUserInfo(info)
	if err != nil {
		http.Error(w, "Failed to set info", http.StatusBadRequest)
		return
	}

	resp, _ := json.Marshal(info)
	w.Write(resp)
}
