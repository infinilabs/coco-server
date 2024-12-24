/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package yuque

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"strings"
)

func get(path, token string) *util.Result {
	req := util.NewGetRequest(util.JoinPath("https://www.yuque.com", path), nil)
	req.AddHeader("X-Auth-Token", token)
	res, err := util.ExecuteRequest(req)
	if err != nil {
		panic(err)
	}

	if res != nil {
		if res.StatusCode > 300 {
			panic(res.Body)
		}
	}

	return res
}

func (this *Plugin) getIconLink(connector,category,iconType string) string {
	// Retrieve the environment's configuration
	env := global.Env()
	tlsEnabled := env.SystemConfig.WebAppConfig.TLSConfig.TLSEnabled
	domain := env.SystemConfig.WebAppConfig.Domain

	// Fallback to publish address if domain is empty
	if domain == "" {
		domain = env.SystemConfig.WebAppConfig.NetworkConfig.GetPublishAddr()
	}

	// Construct the base URL
	protocol := "http"
	if tlsEnabled {
		protocol = "https"
	}
	baseURL := protocol + "://" + domain

	icon:= fmt.Sprintf("%s/assets/connector/%s/%s/%s.png",
			baseURL,this.cleanupIconName(connector),this.cleanupIconName(category),this.cleanupIconName(iconType))

	log.Infof("get icon: %v",icon)

	return icon

		////TODO cache and checking vfs and if not exists and then return the default icon
		//return baseURL + "/assets/connector/default.png"
}

func (this *Plugin) cleanupIconName(name string)string  {
	name=strings.ReplaceAll(name,"/","_")
	name=strings.ReplaceAll(name,"\\","_")
	name=strings.ReplaceAll(name,".","_")
	return strings.ToLower(name)
}


func (this *Plugin) save(obj interface{}) {

	data := util.MustToJSONBytes(obj)

	if global.Env().IsDebug {
		log.Tracef(string(data))
	}
	err := queue.Push(this.cfg.Queue, data)
	if err != nil {
		panic(err)
	}
}

func (this *Plugin) collect() {

	token:=this.cfg.Token

	//for groups only
	res := get("/api/v2/user", token)
	user := struct {
		Group Group `json:"data"`
	}{}
	err := util.FromJSONBytes(res.Body, &user)
	if err != nil {
		panic(err)
	}

	if user.Group.Login == "" {
		panic("invalid group:" + string(res.Body))
	}

	//get users in group, TODO handle pagination
	res = get(fmt.Sprintf("/api/v2/groups/%s/users", user.Group.Login), token)
	users := struct {
		GroupUsers []GroupUser `json:"data"`
	}{}
	err = util.FromJSONBytes(res.Body, &users)
	if err != nil {
		panic(err)
	}

	if this.cfg.IndexingUsers || this.cfg.IndexingGroups {
		//get all users
		for _, user := range users.GroupUsers {
			if user.User != nil && this.cfg.IndexingUsers {
				document := common.Document{
					Source:  YuqueKey,
					Title:   user.User.Name,
					Summary: user.User.Description,
					Type:    user.User.Type,
					URL:     fmt.Sprintf("https://infini.yuque.com/%v", user.User.Login),
					Icon:     user.User.AvatarURL,
					Thumbnail: user.User.AvatarURL,
					Metadata: util.MapStr{
						"user_id":            user.User.ID,
						"user_login":         user.User.Login,
						"public":             user.User.Public,
						"books_count":        user.User.BooksCount,
						"follower_count":     user.User.FollowersCount,
						"following_count":    user.User.FollowingCount,
						"public_books_count": user.User.PublicBooksCount,
					},
				}

				document.ID = util.MD5digest(fmt.Sprintf("%v-%v-%v", "test", "yuque-user", user.User.ID))

				log.Debug("indexing user:", document.ID, user.User.Login, user.User.Name)

				document.Created = &user.User.CreatedAt
				document.Updated = &user.User.UpdatedAt

				this.save(document)

			} else if user.Group != nil && this.cfg.IndexingGroups {
				document := common.Document{
					Source:  YuqueKey,
					Title:   user.Group.Name,
					Summary: user.Group.Description,
					Type:    user.Group.Type,
					URL:     fmt.Sprintf("https://infini.yuque.com/%v", user.Group.ID),
					Icon:      user.Group.AvatarURL,
					Thumbnail: user.Group.AvatarURL,
					Metadata: util.MapStr{
						"user_id":            user.Group.ID,
						"user_login":         user.Group.Login,
						"public":             user.Group.Public,
						"books_count":        user.Group.BooksCount,
						"member_count":       user.Group.MembersCount,
						"public_books_count": user.Group.PublicBooksCount,
					},
				}

				document.ID = util.MD5digest(fmt.Sprintf("%v-%v-%v", "test", "yuque-group", user.Group.ID))

				log.Debug("indexing group:", document.ID, user.Group.Login, user.Group.Name)

				document.Created = &user.Group.CreatedAt
				document.Updated = &user.Group.UpdatedAt

				this.save(document)
			}
		}
	}

	//get all repos, TODO handle pagination
	res = get(fmt.Sprintf("/api/v2/groups/%s/repos", user.Group.Login), token)
	books := struct {
		Books []Book `json:"data"`
	}{}

	err = util.FromJSONBytes(res.Body, &books)
	if err != nil {
		panic(err)
	}

	for _, book := range books.Books {
		//get book details, TODO handle pagination
		res = get(fmt.Sprintf("/api/v2/repos/%v", book.ID), token)
		bookDetail := struct {
			Book BookDetail `json:"data"`
		}{}

		err = util.FromJSONBytes(res.Body, &bookDetail)
		if err != nil {
			panic(err)
		}

		if this.cfg.IndexingBooks && ( bookDetail.Book.Public > 0 || (this.cfg.IncludePrivateBook) ){

			//index repo
			document := common.Document{
				Source:  YuqueKey,
				Title:   bookDetail.Book.Name,
				Summary: bookDetail.Book.Description,
				Type:    book.Type,
				Size:    bookDetail.Book.ItemsCount,
				URL:     fmt.Sprintf("https://infini.yuque.com/infini/%v", bookDetail.Book.Slug),
				Owner: &common.UserInfo{
					UserAvatar: bookDetail.Book.User.AvatarURL,
					UserName:   bookDetail.Book.User.Name,
					UserID:     bookDetail.Book.User.Login,
				},
				Icon: this.getIconLink(YuqueKey,"book",bookDetail.Book.Type),
				//Thumbnail: bookDetail.Book.,
				Metadata: util.MapStr{
					"public":        bookDetail.Book.Public,
					"slug":          bookDetail.Book.Slug,
					"creator_id":    bookDetail.Book.CreatorID,
					"user_id":       bookDetail.Book.UserID,
					"toc_yml":       bookDetail.Book.TocYML,
					"items_count":   bookDetail.Book.ItemsCount,
					"likes_count":   bookDetail.Book.LikesCount,
					"watches_count": bookDetail.Book.WatchesCount,
					"namespace":     bookDetail.Book.Namespace,
					"user":          bookDetail.Book.User,
				},
			}

			document.ID = util.MD5digest(fmt.Sprintf("%v-%v-%v", "test", "yuque-book", bookDetail.Book.ID))

			log.Debug("indexing book:", document.ID, bookDetail.Book.Slug, bookDetail.Book.Namespace, bookDetail.Book.Name)

			document.Created = &bookDetail.Book.CreatedAt
			document.Updated = &bookDetail.Book.UpdatedAt

			this.save(document)
		}

		//get docs in repo, TODO handle pagination
		res = get(fmt.Sprintf("/api/v2/repos/%v/docs?optional_properties=tags,hits,latest_version_id", bookDetail.Book.ID), token)
		doc := struct {
			Meta struct {
				Total int `json:"total"`
			} `json:"meta"`
			Docs []Document `json:"data"`
		}{}

		err = util.FromJSONBytes(res.Body, &doc)
		if err != nil {
			panic(err)
		}

		for _, doc := range doc.Docs {
			//get doc details
			res = get(fmt.Sprintf("/api/v2/repos/%v/docs/%v", bookDetail.Book.ID, doc.ID), token)
			doc := struct {
				Doc DocumentDetail `json:"data"`
			}{}

			err = util.FromJSONBytes(res.Body, &doc)
			if err != nil {
				panic(err)
			}

			if this.cfg.IndexingDocs && (  doc.Doc.Public > 0 || (this.cfg.IncludePrivateDoc) ){
				//index doc
				document := common.Document{
					Source:  YuqueKey,
					Title:   doc.Doc.Title,
					Summary: doc.Doc.Description,
					Cover:   doc.Doc.Cover,
					Content: doc.Doc.Body,
					Type:    doc.Doc.Type,
					Size:    doc.Doc.WordCount,
					URL:     fmt.Sprintf("https://infini.yuque.com/go/doc/%v", doc.Doc.ID),
					Owner: &common.UserInfo{
						UserAvatar: doc.Doc.User.AvatarURL,
						UserName:   doc.Doc.User.Name,
						UserID:     doc.Doc.User.Login,
					},
					Icon:      this.getIconLink(YuqueKey,"doc",doc.Doc.Type),
					Thumbnail: doc.Doc.Cover,
					Metadata: util.MapStr{
						"public":         doc.Doc.Public,
						"slug":           doc.Doc.Slug,
						"user_id":        doc.Doc.UserID,
						"book_id":        doc.Doc.BookID,
						"last_editor_id": doc.Doc.LastEditorID,
						"format":         doc.Doc.Format,
						"body_draft":     doc.Doc.BodyDraft,
						"body_html":      doc.Doc.BodyHTML,
						"body_sheet":     doc.Doc.BodySheet,
						"body_lake":      doc.Doc.BodyLake,
						"body_table":     doc.Doc.BodyTable,
						"status":         doc.Doc.Status,
						"likes_count":    doc.Doc.LikesCount,
						"read_count":     doc.Doc.ReadCount,
						"comments_count": doc.Doc.CommentsCount,
						"word_count":     doc.Doc.WordCount,
						"user":           doc.Doc.User,
						"creator":        doc.Doc.Creator,
						"book":           doc.Doc.Book,
						"tags":           doc.Doc.Tags,
					},
				}

				document.ID = util.MD5digest(fmt.Sprintf("%v-%v-%v", "test", "yuque-doc", doc.Doc.ID))

				log.Debug("indexing doc:", document.ID, doc.Doc.Slug, doc.Doc.Title, doc.Doc.WordCount)

				document.Created = &doc.Doc.CreatedAt
				document.Updated = &doc.Doc.UpdatedAt

				this.save(document)
			}
		}
	}
}
