# Dropbox Connector

This connector allows you to index files from Dropbox.

## Configuration

To use this connector, you need to create a Dropbox App in the [Dropbox App Console](https://www.dropbox.com/developers/apps).

### Steps

1. Go to [Dropbox App Console](https://www.dropbox.com/developers/apps).
2. Click "Create app".
3. Choose "Scoped access".
4. Choose "Full Dropbox" or "App folder" depending on your needs.
5. Name your app.
6. In the "Permissions" tab, enable the following permissions:
   - `files.content.read`
   - `files.metadata.read`
   - `sharing.read`
   - `account_info.read`
   - `team_data.member` (if using team features)
7. In the "Settings" tab, add the Redirect URI:
   - `http://localhost:8080/connector/{connector_id}/dropbox/oauth_redirect`
   - Replace `http://localhost:8080` with your server address.
   - Note: The `{connector_id}` part is dynamic. You might need to add the exact URL when you know the connector ID, or use a wildcard if supported (Dropbox requires exact match).
   - Actually, Coco Server usually handles the redirect URI dynamically or you configure it.
   - For Coco Server, the redirect URL is typically constructed by the frontend.
   
   Default Redirect URI format: `YOUR_COCO_SERVER_URL/connector/{connector_id}/oauth_redirect` (Check the code for exact path).
   
   In `processor.go`:
   `api.HandleUIMethod(api.GET, "/connector/:id/dropbox/oauth_redirect", oAuthRedirect, api.RequireLogin())`
   
   So the URL path is `/connector/:id/dropbox/oauth_redirect` .
   
8. Copy "App key" and "App secret".

### Connector Settings

- **App Key**: The App key from Dropbox.
- **App Secret**: The App secret from Dropbox.
- **Path** (Optional): The folder path to index (e.g. `/MyFolder`). Defaults to root if empty.

