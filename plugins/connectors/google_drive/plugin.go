/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package google_drive

import (
	"context"
	"errors"
	"infini.sh/framework/core/pipeline"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"golang.org/x/oauth2"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

type Credential struct {
	ClientId     string `config:"client_id" json:"client_id"`
	AuthUri      string `config:"auth_url" json:"auth_url"`
	TokenUri     string `config:"token_url" json:"token_url"`
	ClientSecret string `config:"client_secret" json:"client_secret"`
	RedirectUri  string `config:"redirect_url" json:"redirect_url"`
}

type Config struct {
	AccessToken  string      `config:"access_token" json:"access_token"`
	RefreshToken string      `config:"refresh_token" json:"refresh_token"`
	TokenExpiry  string      `config:"token_expiry" json:"token_expiry"`
	Profile      util.MapStr `config:"profile" json:"profile"`
}

func (this *Processor) Fetch(pipeCtx *pipeline.Context, connector *common.Connector, datasource *common.DataSource) error {
	config := Config{}
	this.MustParseConfig(datasource, &config)

	oAuthConfig := getOAuthConfig(connector.ID)
	if oAuthConfig == nil {
		return errors.New("invalid oauth config")
	}

	log.Tracef("handle google_drive's datasource: %v", config)

	if config.AccessToken != "" && config.RefreshToken != "" {
		// Initialize the token using AccessToken and RefreshToken
		tok := oauth2.Token{
			AccessToken:  config.AccessToken,
			RefreshToken: config.RefreshToken,
			Expiry:       parseExpiry(config.TokenExpiry),
		}

		//log.Debugf("OAuth2 Config: %v", util.MustToJSON(oAuthConfig))
		//log.Debugf("Current Token Config: %v", util.MustToJSON(config))

		// Check if the token is valid
		if !tok.Valid() {
			// Check if SkipInvalidToken is false, which means token must be valid
			if !this.SkipInvalidToken && !tok.Valid() {
				return errors.New("token is invalid")
			}

			// If the token is invalid, attempt to refresh it
			log.Warnf("google drive [%v](%v) token is invalid or expired, attempting to refresh...", datasource.Name, datasource.ID)

			// Refresh the token using the refresh token
			if tok.RefreshToken != "" {
				log.Debug("Attempting to refresh the token")
				log.Debugf("Current token expiry: %v", tok.Expiry)
				log.Debugf("Refresh token present: %v", tok.RefreshToken != "")

				tokenSource := oAuthConfig.TokenSource(context.Background(), &tok)
				refreshedToken, err := tokenSource.Token()
				if err != nil {
					log.Errorf("Failed to refresh token: %v", err)
					// Check for specific OAuth2 errors that indicate invalid refresh token
					if strings.Contains(err.Error(), "unauthorized_client") {
						log.Errorf("Refresh token is invalid or expired, need to re-authorize. This can happen if:")
						log.Errorf("1. The OAuth app client secret has changed")
						log.Errorf("2. The refresh token has been revoked")
						log.Errorf("3. The OAuth app configuration has changed")
						log.Errorf("Please re-authorize the Google Drive connector to obtain a new refresh token.")
						return errors.New("Refresh token is invalid, please re-authorize the Google Drive connector by visiting the connector settings and reconnecting")
					}
					return errors.New("Failed to refresh token")
				}

				// Save the refreshed token
				config.AccessToken = refreshedToken.AccessToken
				config.RefreshToken = refreshedToken.RefreshToken
				config.TokenExpiry = refreshedToken.Expiry.Format(time.RFC3339) // Format using RFC3339

				datasource.Connector.Config = config

				log.Debugf("updating datasource with new refresh token: %v", datasource.ID)

				ctx := orm.NewContext().DirectAccess()

				// Optionally, save the new tokens in your store (e.g., database or config)
				err = orm.Update(ctx, datasource)
				if err != nil {
					log.Errorf("Failed to save updated datasource configuration: %v", err)
					return errors.New("Failed to save updated configuration")
				}

				log.Debug("Token refreshed successfully")
			} else {
				return errors.New("No refresh token available, unable to refresh token.")
			}

		}

		// Token is valid, proceed with indexing files
		log.Debug("start processing google drive files")
		this.startIndexingFiles(pipeCtx, connector, datasource, &tok)
		log.Debug("finished processing google drive files")
	}

	return nil
}

// Helper function to parse token expiry time
func parseExpiry(expiryStr string) time.Time {
	// Parse the expiry time string into a time.Time object
	expiry, err := time.Parse(time.RFC3339, expiryStr)
	if err != nil {
		log.Errorf("Failed to parse token expiry: %v", err)
		return time.Time{} // Return zero value time on error
	}
	return expiry
}
