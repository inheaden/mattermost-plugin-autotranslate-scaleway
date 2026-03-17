package main

import (
	"encoding/json"
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
)

func TestSetCachedTranslation(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.API = api
	plugin.setConfiguration(&configuration{
		ScalewayModel:        "model-a",
		ScalewaySystemPrompt: "prompt-a",
		EnableCache:          true,
		CacheTTLSeconds:      123,
	})

	translated := &TranslatedMessage{
		ID:             "translation-id",
		PostID:         "postid12345678901234567890",
		SourceLanguage: "de",
		SourceText:     "Hallo",
		TargetLanguage: "en",
		TranslatedText: "Hello",
		UpdateAt:       42,
	}

	expectedKey := buildTranslationCacheKey(translated.PostID, translated.UpdateAt, translated.SourceLanguage, translated.TargetLanguage, plugin.getConfiguration())
	expectedEntryBytes, err := json.Marshal(translationCacheEntry{TranslatedMessage: *translated})
	if err != nil {
		t.Fatalf("failed to marshal expected cache entry: %v", err)
	}
	api.On("KVSetWithExpiry", expectedKey, expectedEntryBytes, int64(123)).Return(nil)

	err = plugin.setCachedTranslation(translated)
	if err != nil {
		t.Fatalf("setCachedTranslation returned error: %v", err)
	}
	api.AssertExpectations(t)
}

func TestGetCachedTranslation(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.API = api
	plugin.setConfiguration(&configuration{
		ScalewayModel:        "model-a",
		ScalewaySystemPrompt: "prompt-a",
		EnableCache:          true,
	})

	expected := TranslatedMessage{
		ID:             "translation-id",
		PostID:         "postid12345678901234567890",
		SourceLanguage: "de",
		SourceText:     "Hallo",
		TargetLanguage: "en",
		TranslatedText: "Hello",
		UpdateAt:       42,
	}

	cacheKey := buildTranslationCacheKey(expected.PostID, expected.UpdateAt, expected.SourceLanguage, expected.TargetLanguage, plugin.getConfiguration())
	entryBytes, err := json.Marshal(translationCacheEntry{TranslatedMessage: expected})
	if err != nil {
		t.Fatalf("failed to marshal cache entry: %v", err)
	}

	api.On("KVGet", cacheKey).Return(entryBytes, (*model.AppError)(nil))

	actual, err := plugin.getCachedTranslation(&expected)
	if err != nil {
		t.Fatalf("getCachedTranslation returned error: %v", err)
	}
	if actual == nil {
		t.Fatal("expected cached translation, got nil")
	}
	if *actual != expected {
		t.Fatalf("cached translation mismatch: got %#v want %#v", *actual, expected)
	}
	api.AssertExpectations(t)
}

func TestTranslatePostReturnsCachedTranslation(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.API = api
	plugin.setConfiguration(&configuration{
		ScalewayModel:        "model-a",
		ScalewaySystemPrompt: "prompt-a",
		EnableCache:          true,
	})

	post := &model.Post{
		Id:       "postid12345678901234567890",
		Message:  "Hallo",
		UpdateAt: 42,
	}

	expected := &TranslatedMessage{
		ID:             post.Id + "de" + "en" + "42",
		PostID:         post.Id,
		SourceLanguage: "de",
		SourceText:     post.Message,
		TargetLanguage: "en",
		TranslatedText: "Hello",
		UpdateAt:       post.UpdateAt,
	}

	cacheKey := buildTranslationCacheKey(post.Id, post.UpdateAt, "de", "en", plugin.getConfiguration())
	entryBytes, err := json.Marshal(translationCacheEntry{TranslatedMessage: *expected})
	if err != nil {
		t.Fatalf("failed to marshal cache entry: %v", err)
	}

	api.On("KVGet", cacheKey).Return(entryBytes, (*model.AppError)(nil))

	actual, err := plugin.translatePost(post, "de", "en")
	if err != nil {
		t.Fatalf("translatePost returned error: %v", err)
	}
	if actual == nil {
		t.Fatal("expected cached translation, got nil")
	}
	if *actual != *expected {
		t.Fatalf("translated message mismatch: got %#v want %#v", *actual, *expected)
	}
	api.AssertExpectations(t)
}
