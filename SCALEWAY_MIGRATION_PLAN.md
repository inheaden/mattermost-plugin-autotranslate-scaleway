# Migration Plan: Amazon Translate to Scaleway Inference API

## Goal

Replace the current Amazon Translate integration with a call to Scaleway's Generative APIs chat-completions endpoint, while keeping the user-facing Mattermost workflow as stable as possible.

## Summary

- Amazon Translate is a dedicated translation API.
- Scaleway's equivalent integration path is an OpenAI-compatible chat-completions API.
- Translation will be implemented through prompting instead of AWS-native `source`, `target`, and `text` request fields.
- To improve reliability, the plugin should request structured JSON output from the model instead of parsing free-form text.

## Verified Scaleway API Details

### Endpoint and authentication

- Base URL: `https://api.scaleway.ai/v1`
- Project-scoped base URL: `https://api.scaleway.ai/<PROJECT_ID>/v1`
- Chat endpoint: `POST /chat/completions`
- Auth header: `Authorization: Bearer <SCW_SECRET_KEY>`
- Content type: `application/json`

### Structured output support

Scaleway supports OpenAI-style structured output:

- JSON mode: `"response_format": {"type": "json_object"}`
- JSON schema mode: `"response_format": {"type": "json_schema", "json_schema": {...}}`

For this plugin, `json_schema` is the safest option because it lets the backend enforce a predictable response shape.

## Recommended Request Shape

Use a deterministic translation prompt with low randomness:

```json
{
  "model": "mistral-small-3.2-24b-instruct-2506",
  "temperature": 0,
  "messages": [
    {
      "role": "system",
      "content": "You are a translation engine. Preserve meaning, tone, markdown, mentions, URLs, and line breaks. Return only valid JSON."
    },
    {
      "role": "user",
      "content": "Translate the following text from auto to English: Bonjour tout le monde"
    }
  ],
  "response_format": {
    "type": "json_schema",
    "json_schema": {
      "name": "translation_result",
      "schema": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "translated_text": {
            "type": "string"
          },
          "detected_source_language": {
            "type": "string"
          }
        },
        "required": [
          "translated_text"
        ]
      }
    }
  }
}
```

## Configuration Migration

### Remove

The current AWS-specific settings can be removed:

- `AWSAccessKeyID`
- `AWSSecretAccessKey`
- `AWSRegion`

### Add

Introduce the following plugin settings:

- `ScalewaySecretKey`
  - Required
  - Used in the `Authorization` header
- `ScalewayProjectID`
  - Optional
  - Allows project-scoped API URL construction
- `ScalewayModel`
  - Required
  - Example: `mistral-small-3.2-24b-instruct-2506`
- `ScalewayBaseURL`
  - Optional
  - Default: `https://api.scaleway.ai/v1`
- `ScalewayTemperature`
  - Optional
  - Default: `0`
- `ScalewayMaxTokens`
  - Optional
  - Used to bound response size
- `ScalewaySystemPrompt`
  - Optional
  - Lets admins customize translation behavior without code changes

### Validation rules

Update configuration validation to require:

- non-empty `ScalewaySecretKey`
- non-empty `ScalewayModel`
- default `ScalewayBaseURL` if unset
- default `ScalewayTemperature` of `0` if unset

`ScalewayProjectID`, `ScalewayMaxTokens`, and `ScalewaySystemPrompt` can remain optional.

## Backend Migration Plan

### 1. Replace AWS SDK usage

Current implementation in `server/api.go`:

- creates an AWS session
- builds AWS static credentials
- calls `translate.Text(...)`

Replace this with:

- a plain HTTP client
- a JSON request body for Scaleway chat completions
- JSON response parsing

This change also allows dropping AWS SDK dependencies from the server module.

### 2. Add request and response models

Add internal Go structs for:

- chat completion request
- chat message
- response format schema
- chat completion response
- parsed translation payload

Keep these local to the server package and avoid introducing a new SDK unless there is a strong reason to do so.

### 3. Convert current user language settings into prompt input

The plugin already stores:

- `source_language`
- `target_language`

These should remain as the user-facing settings to preserve the slash-command behavior.

Convert them into prompt text:

- if source is `auto`: ask the model to detect the source language automatically
- otherwise: explicitly instruct the model to translate from source to target

## Language Handling Plan

### Current state

The plugin currently validates against AWS-supported language codes in `server/command.go` and `server/plugin.go`.

### Recommended approach

Keep the current language-code UX initially, but decouple it from AWS wording and assumptions.

Implement a local mapping layer:

- `en` -> `English`
- `de` -> `German`
- `fr` -> `French`
- `auto` -> automatic detection

Use language names in prompts instead of short codes. This is more robust for model-driven translation.

### Follow-up option

If needed later, relax validation to a plugin-maintained list that is no longer described as "AWS supported languages".

## Error Handling Changes

Update server-side error handling for these cases:

- missing or invalid Scaleway credentials
- non-200 response from Scaleway
- missing `choices[0].message.content`
- invalid JSON returned by the model
- empty `translated_text`

Return stable plugin errors instead of exposing raw provider errors directly to users where possible.

## Suggested Code Changes

### Files to update

- `server/api.go`
  - Replace AWS Translate call with HTTP call to Scaleway
- `server/configuration.go`
  - Replace AWS config fields and validation
- `plugin.json`
  - Replace system-console settings
- `server/command.go`
  - Update help text and provider references
- `webapp/src/manifest.js`
  - Regenerated via `make apply` after `plugin.json` changes
- `server/manifest.go`
  - Regenerated via `make apply` after `plugin.json` changes
- `README.md`
  - Update setup documentation and provider references

### Dependencies

Expected cleanup after migration:

- remove `github.com/aws/aws-sdk-go` from the server module if no longer used elsewhere

## Rollout Strategy

### Phase 1

- Replace provider configuration
- Implement Scaleway request path
- Keep current slash commands and user settings unchanged
- Keep existing translated message response shape unchanged for the webapp

### Phase 2

- Update help text and README
- Rename provider-specific classes and labels where helpful
- Review whether language validation should be broadened beyond the current AWS-based list

## Risks

- Model output can still vary if prompts are weak or if structured output is not enforced
- Language detection quality may differ from Amazon Translate
- Some AWS language-code assumptions may not map cleanly to model prompting
- Long posts may require token budgeting or truncation rules

## Recommended Implementation Decisions

- Use plain HTTP instead of adding a Scaleway SDK
- Use `json_schema` structured output instead of free-form text parsing
- Keep slash commands and stored user preferences stable in the first migration
- Convert language codes to human-readable names before prompting
- Set `temperature` to `0`

## Suggested Example Response Contract

Internal response parsed from the model:

```json
{
  "translated_text": "Hello world",
  "detected_source_language": "fr"
}
```

Plugin response returned to the webapp should stay compatible with the current shape:

```json
{
  "id": "postidautoen123456789",
  "post_id": "postid",
  "source_lang": "auto",
  "source_text": "Bonjour tout le monde",
  "target_lang": "en",
  "translated_text": "Hello world",
  "update_at": 123456789
}
```

## Sources

- Scaleway Documentation Platform: `generative-apis/api-cli/using-generative-apis`
- Scaleway Documentation Platform: `generative-apis/api-cli/using-chat-api`
- Scaleway Documentation Platform: `generative-apis/how-to/use-structured-outputs`
- Scaleway Documentation Platform: `managed-inference/concepts`
- Context7 library: `/scaleway/docs-content`
