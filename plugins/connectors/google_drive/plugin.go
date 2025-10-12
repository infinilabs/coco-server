/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package google_drive

import (
	"context"
	"time"

	log "github.com/cihub/seelog"
	"golang.org/x/oauth2"
	"infini.sh/coco/modules/common"
	config3 "infini.sh/framework/core/config"
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

func (this *Processor) fetch_google_drive(connector *common.Connector, datasource *common.DataSource) {
	if connector == nil || datasource == nil {
		panic("invalid connector config")
	}

	cfg, err := config3.NewConfigFrom(datasource.Connector.Config)
	if err != nil {
		panic(err)
	}

	datasourceCfg := Config{}
	err = cfg.Unpack(&datasourceCfg)
	if err != nil {
		panic(err)
	}

	oAuthConfig := getOAuthConfig(connector.ID)
	if oAuthConfig == nil {
		panic("invalid oauth config")
	}

	log.Tracef("handle google_drive's datasource: %v", datasourceCfg)

	if datasourceCfg.AccessToken != "" && datasourceCfg.RefreshToken != "" {
		// Initialize the token using AccessToken and RefreshToken
		tok := oauth2.Token{
			AccessToken:  datasourceCfg.AccessToken,
			RefreshToken: datasourceCfg.RefreshToken,
			Expiry:       parseExpiry(datasourceCfg.TokenExpiry),
		}

		// Check if the token is valid
		if !tok.Valid() {
			// Check if SkipInvalidToken is false, which means token must be valid
			if !this.SkipInvalidToken && !tok.Valid() {
				panic("token is invalid")
			}

			// If the token is invalid, attempt to refresh it
			log.Warnf("Token is invalid or expired, attempting to refresh...")

			// Refresh the token using the refresh token
			if tok.RefreshToken != "" {
				log.Debug("Attempting to refresh the token")
				tokenSource := oAuthConfig.TokenSource(context.Background(), &tok)
				refreshedToken, err := tokenSource.Token()
				if err != nil {
					log.Errorf("Failed to refresh token: %v", err)
					panic("Failed to refresh token")
				}

				// Save the refreshed token
				datasourceCfg.AccessToken = refreshedToken.AccessToken
				datasourceCfg.RefreshToken = refreshedToken.RefreshToken
				datasourceCfg.TokenExpiry = refreshedToken.Expiry.Format(time.RFC3339) // Format using RFC3339

				datasource.Connector.Config = datasourceCfg

				log.Debugf("updating datasource with new refresh token: %v", datasource.ID)

				ctx := orm.NewContext().DirectAccess()

				// Optionally, save the new tokens in your store (e.g., database or config)
				err = orm.Update(ctx, datasource)
				if err != nil {
					log.Errorf("Failed to save updated datasource configuration: %v", err)
					panic("Failed to save updated configuration")
				}

				log.Debug("Token refreshed successfully")
			} else {
				log.Warnf("No refresh token available, unable to refresh token.")
			}

		} else {
			// Token is valid, proceed with indexing files
			log.Debug("start processing google drive files")
			this.startIndexingFiles(connector, datasource, &tok)
			log.Debug("finished processing google drive files")
		}
	}

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
