# Introduction
`LLN` is a twitterlike api server 

## API documentation

### Authorization
| Method | Path        | Description |
| ------ | ----------- |-------------|
| POST | /o/authorize/{oidc-provider} | authorize use oidc `code` |
| GET | /o/authorize/{oidc-provider} | authorize use oidc `code` and redirect |
| GET | /o/oidc/{oidc-provider} | redirect to oidc provider for authorize  |

### Status
| Method | Path        | Description |
| ------ | ----------- |-------------|
| POST | /i/status                  | Post new status      |
| POST | /i/like/status/{status-id} | Like status          |
| POST | /i/bookmark/status/{status-id} | Bookmark status  |
| GET  | /i/bookmarks | List bookmark status |  
| GET  | /o/status/{status-id}      | Status details |  
| GET  | /o/status/{status-id}/comments | Status comments |
| GET  | /o/user/{unique-name}/status | Get user status   |
| GET  | /o/explore | Explore status |
| GET  | /o/labels | List labels |

### Messages
| Method | Path        | Description |
| ------ | ----------- |-------------|
| GET | /i/messages                  | Message list      |
| GET | /i/messages/tips | New messages          |
| DELETE | /i/messages/tips | Mark messages read         |

### User
| Method | Path        | Description |
| ------ | ----------- |-------------|
| POST | /i/like/user/{unique-name}  | Like user        |
| PUT | /i/name                     | Change user name  |
| GET | /o/user/{unique-name}       | Get user profile  |
