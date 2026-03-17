package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

const translationCacheKeyPrefix = "translation:"

type translationCacheEntry struct {
	TranslatedMessage
}

func buildTranslationCacheKey(postID string, updateAt int64, source, target string, configuration *configuration) string {
	promptHash := sha256.Sum256([]byte(configuration.getScalewaySystemPrompt()))

	return fmt.Sprintf(
		"%s%s:%d:%s:%s:%s:%s",
		translationCacheKeyPrefix,
		postID,
		updateAt,
		source,
		target,
		configuration.getScalewayModel(),
		hex.EncodeToString(promptHash[:]),
	)
}

func (p *Plugin) getCachedTranslation(post *TranslatedMessage) (*TranslatedMessage, error) {
	configuration := p.getConfiguration()
	if !configuration.isCacheEnabled() {
		return nil, nil
	}

	cacheKey := buildTranslationCacheKey(post.PostID, post.UpdateAt, post.SourceLanguage, post.TargetLanguage, configuration)
	entryBytes, appErr := p.API.KVGet(cacheKey)
	if appErr != nil {
		return nil, appErr
	}
	if entryBytes == nil {
		return nil, nil
	}

	var entry translationCacheEntry
	if err := json.Unmarshal(entryBytes, &entry); err != nil {
		return nil, err
	}

	translated := entry.TranslatedMessage
	return &translated, nil
}

func (p *Plugin) setCachedTranslation(translated *TranslatedMessage) error {
	configuration := p.getConfiguration()
	if !configuration.isCacheEnabled() {
		return nil
	}

	entryBytes, err := json.Marshal(translationCacheEntry{
		TranslatedMessage: *translated,
	})
	if err != nil {
		return err
	}

	cacheKey := buildTranslationCacheKey(translated.PostID, translated.UpdateAt, translated.SourceLanguage, translated.TargetLanguage, configuration)
	if appErr := p.API.KVSetWithExpiry(cacheKey, entryBytes, int64(configuration.getCacheTTLSeconds())); appErr != nil {
		return appErr
	}

	return nil
}
