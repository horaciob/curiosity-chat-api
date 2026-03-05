# iOS Senior Dev — Persistent Memory

## Active Project: Whisplorer

App source root: `/Users/horaciobranciforte/labs/curiosity/whisplorer/whisplorer/`

### Architecture
- SwiftUI, iOS 26+, Swift 5, `@Observable @MainActor` ViewModels
- MVVM pattern throughout; no TCA or Composable Architecture
- `URLSession.shared` used directly — no custom session or interceptor layer
- All REST responses use JSON:API format (`application/vnd.api+json`)
- No Combine; concurrency is `async/await` with `TaskGroup` for parallelism

### Key Files
- `Config.swift` — single source of truth for all base URLs
- `Services/APIClient.swift` — all non-chat REST calls; split across two base URLs
- `Services/ChatService.swift` — chat REST + token-injection helper
- `Services/UserSession.swift` — session state, token storage, token fetch/refresh
- `Views/Chat/ChatViewModel.swift` — WebSocket lifecycle

### Auth & Token Strategy
- No SSO; userId stored in UserDefaults; email-based find-or-create flow
- On login/signup: `POST /internal/token {"user_id":"<uuid>"}` via `http://localhost:8084`
  with `X-Internal-Key: <Config.UserAPI.internalKey>` — returns `{access_token, refresh_token}`
- Tokens stored in `UserDefaults` (keys: `chatToken`, `chatRefreshToken`)
- Token refresh: `POST /api/v1/auth/refresh {"refresh_token":"..."}` via `Config.UserAPI.baseURL`
- `ChatService.perform()` auto-retries once on 401 after refreshing token
- Reaction endpoints on POI-api require `Authorization: Bearer <token>` — passed via `token` param
- User-api write endpoints (favorites, visits, follows) are unauthenticated in current router

### Config / Base URL Pattern
- All URLs are static constants in `Config.swift` nested enums
- `Config.UserAPI.baseURL`  → `http://localhost:8084/api/v1` (users, auth, follows, favorites, visits)
- `Config.UserAPI.internalKey` → `"dev-internal-key"` (must match INTERNAL_API_KEY env var)
- `Config.POIAPI.baseURL`   → `http://localhost:8083/api/v1` (POIs, categories, reactions)
- `Config.ChatAPI.baseURL`  → `http://localhost:8081/api/v1`
- `Config.ChatAPI.wsURL`    → `ws://localhost:8081/api/v1/ws`
- Old constants (`CuriosityAPI`, `CuriosityAuth`, `CuriosityChat`) are REMOVED

### APIClient Routing
- `APIClient.userBaseURL` = `Config.UserAPI.baseURL` — /users/*, /categories via user-api
- `APIClient.poiBaseURL`  = `Config.POIAPI.baseURL` — /pois/*, /categories
- User endpoints: findUser, createUser, validateUser, searchUsers, savePreferences,
  registerDevice, addFavorite, removeFavorite, fetchFavoritePoiIDs, recordVisit,
  fetchVisitedPoiIDs, followUser, unfollowUser, fetchFollowing, fetchFollowers,
  fetchPendingFollowRequests, acceptFollowRequest, rejectFollowRequest
- POI endpoints: fetchNearbyPOIs, fetchTranslation, fetchCategories, searchPOIs,
  fetchReaction, reactToPOI, removeReaction

### Favorites Response Shape (new user-api)
- `GET /users/{id}/favorites` returns `type: "favorites"` with `{user_id, poi_id, created_at}`
- Decoded with `FavoriteAttributes` struct (in `Models/POI.swift`) — NOT `POIAttributes`
- `fetchFavoritePoiIDs` extracts `$0.attributes.poiId` from each resource

### Reaction Token Injection
- `fetchReaction`, `reactToPOI`, `removeReaction` all take `token: String` parameter
- `POIDetailView` has `var token: String = ""` — passed from `MapView` and `SearchView` as `session.chatToken`

### WebSocket
- Opened in `ChatViewModel.connectWebSocket(token:)`
- URL from `ChatService.wsURL` (port 8081, unchanged)
- Auth sent as first text frame: `{"type":"auth","token":"<jwt>"}`
- Receives `auth_ok` ack frame, then message frames
- Disconnect: `viewModel.disconnectWebSocket()` on `.onDisappear`

### Pagination
- All paginated calls use `page[limit]` / `page[offset]` query params
- `APIClient.pageSize = 50` (from `Config.Pagination.pageSize`)
- Nearby POI meta only returns `total`; categories/users return full meta

### Notable Dead Code
- `APIClient.fetchFollowers(userID:)` is defined but never called from any view or VM
