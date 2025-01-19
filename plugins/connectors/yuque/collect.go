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

func (this *Plugin) getIconKey(category, iconType string) string {
	return strings.TrimSpace(strings.ToLower(iconType))
}

func (this *Plugin) cleanupIconName(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, ".", "_")
	return strings.ToLower(name)
}

func (this *Plugin) save(obj interface{}) {

	data := util.MustToJSONBytes(obj)

	if global.Env().IsDebug {
		log.Tracef(string(data))
	}
	err := queue.Push(this.Queue, data)
	if err != nil {
		panic(err)
	}
}

func (this *Plugin) collect(cfg *YuqueConfig) {

	token := cfg.Token

	if token == "" {
		panic("invalid yuque token")
	}

	//for groups only
	res := get("/api/v2/user", token)
	currentUser := struct {
		Group Group `json:"data"`
	}{}

	err := util.FromJSONBytes(res.Body, &currentUser)
	if err != nil {
		panic(err)
	}

	if currentUser.Group.Login == "" {
		panic("invalid group:" + string(res.Body))
	}

	log.Infof("start collecting for %v", currentUser.Group.Login)

	//get users in group
	if cfg.IndexingUsers || cfg.IndexingGroups {
		this.collectUsers(currentUser.Group.Login, token, cfg)
	}

	//get all books
	if cfg.IndexingBooks || cfg.IndexingDocs {
		this.collectBooks(currentUser.Group.Login, token, cfg)
	}

	log.Infof("finished collecting for %v", currentUser.Group.Login)
}

func (this *Plugin) collectBooks(login, token string, cfg *YuqueConfig) {

	const limit = 100
	offset := 0

	for {
		res := get(fmt.Sprintf("/api/v2/groups/%s/repos?offse=%v&limit=%v", login, offset, limit), token)
		books := struct {
			Books []Book `json:"data"`
		}{}

		err := util.FromJSONBytes(res.Body, &books)
		if err != nil {
			panic(err)
		}

		log.Infof("fetched %v books for %v, offset: %v", len(books.Books), login, offset)

		for _, book := range books.Books {
			//get book details
			res = get(fmt.Sprintf("/api/v2/repos/%v", book.ID), token)
			bookDetail := struct {
				Book BookDetail `json:"data"`
			}{}

			err = util.FromJSONBytes(res.Body, &bookDetail)
			if err != nil {
				panic(err)
			}

			bookID := bookDetail.Book.ID

			if cfg.IndexingBooks && (bookDetail.Book.Public > 0 || (cfg.IncludePrivateBook)) {

				//index books
				document := common.Document{
					Source: common.DataSourceReference{
						//ID: "",//TODO
						Name: YuqueKey,
						Type: "connector",
					},
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
					Icon: this.getIconKey("book", bookDetail.Book.Type),
					//Thumbnail: bookDetail.Book.,
				}

				document.Metadata = util.MapStr{
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
				}

				document.ID = util.MD5digest(fmt.Sprintf("%v-%v-%v", "test", "yuque-book", bookID))

				log.Debug("indexing book: %v, %v, %v, %v", document.ID, bookDetail.Book.Slug, bookDetail.Book.Namespace, bookDetail.Book.Name)

				document.Created = &bookDetail.Book.CreatedAt
				document.Updated = &bookDetail.Book.UpdatedAt

				this.save(document)
			} else {
				log.Debug("skip book:", bookDetail.Book.Name, ",", bookDetail.Book.Public)
			}

			//get docs in repo
			if cfg.IndexingDocs {
				this.collectDocs(login, bookID, token, cfg)
			}
		}

		// Exit loop if no more pages
		if len(books.Books) < limit || global.ShuttingDown() {
			break
		}
		offset += limit
	}

}

func (this *Plugin) collectDocs(login string, bookID int64, token string, cfg *YuqueConfig) {

	const limit = 100
	offset := 0

	for {

		res := get(fmt.Sprintf("/api/v2/repos/%v/docs?offse=%v&limit=%v&optional_properties=tags,hits,latest_version_id", bookID, offset, limit), token)
		doc := struct {
			Meta struct {
				Total int `json:"total"`
			} `json:"meta"`
			Docs []Document `json:"data"`
		}{}

		err := util.FromJSONBytes(res.Body, &doc)
		if err != nil {
			panic(err)
		}

		log.Infof("fetched %v docs for %v, book: %v, offset: %v, total: %v", len(doc.Docs), login, bookID, offset, doc.Meta.Total)

		for _, doc := range doc.Docs {
			if cfg.IndexingDocs && (doc.Public > 0 || (cfg.IncludePrivateDoc)) {
				//get doc details
				this.collectDocDetails(bookID, doc.ID, token, cfg)
			} else {
				log.Debug("skip doc:", doc.Title, ",", doc.Public)
			}
		}

		// Exit loop if no more pages
		if len(doc.Docs) < limit || offset > doc.Meta.Total || global.ShuttingDown() {
			break
		}
		offset += limit
	}

}

func (this *Plugin) collectDocDetails(bookID int64, docID int64, token string, cfg *YuqueConfig) {

	res := get(fmt.Sprintf("/api/v2/repos/%v/docs/%v", bookID, docID), token)
	doc := struct {
		Doc DocumentDetail `json:"data"`
	}{}

	err := util.FromJSONBytes(res.Body, &doc)
	if err != nil {
		panic(err)
	}

	if cfg.IndexingDocs && (doc.Doc.Public > 0 || (cfg.IncludePrivateDoc)) {
		//index doc
		document := common.Document{
			Source: common.DataSourceReference{
				//ID: "",//TODO
				Name: YuqueKey,
				Type: "connector",
			},
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
			Icon:      this.getIconKey("doc", doc.Doc.Type),
			Thumbnail: doc.Doc.Cover,
		}

		document.Metadata = util.MapStr{
			"public":         doc.Doc.Public,
			"slug":           doc.Doc.Slug,
			"user_id":        doc.Doc.UserID,
			"book_id":        doc.Doc.BookID,
			"last_editor_id": doc.Doc.LastEditorID,
			"format":         doc.Doc.Format,
			"status":         doc.Doc.Status,
			"likes_count":    doc.Doc.LikesCount,
			"read_count":     doc.Doc.ReadCount,
			"comments_count": doc.Doc.CommentsCount,
			"word_count":     doc.Doc.WordCount,
			"user":           doc.Doc.User,
			"creator":        doc.Doc.Creator,
			"book":           doc.Doc.Book,
			"tags":           doc.Doc.Tags,
		}

		document.Payload = util.MapStr{
			"body_draft": doc.Doc.BodyDraft,
			"body_html":  doc.Doc.BodyHTML,
			"body_sheet": doc.Doc.BodySheet,
			"body_lake":  doc.Doc.BodyLake,
			"body_table": doc.Doc.BodyTable,
		}

		document.ID = util.MD5digest(fmt.Sprintf("%v-%v-%v", "test", "yuque-doc", doc.Doc.ID))

		log.Debugf("indexing doc: %v, %v, %v, %v", document.ID, doc.Doc.Slug, doc.Doc.Title, doc.Doc.WordCount)

		document.Created = &doc.Doc.CreatedAt
		document.Updated = &doc.Doc.UpdatedAt

		this.save(document)
	}
}

func (this *Plugin) collectUsers(login, token string, cfg *YuqueConfig) {
	const pageSize = 100
	offset := 0

	for {
		// Fetch users in the current group with pagination
		res := get(fmt.Sprintf("/api/v2/groups/%s/users?offset=%d", login, offset), token)
		var users struct {
			GroupUsers []GroupUser `json:"data"`
		}

		if err := util.FromJSONBytes(res.Body, &users); err != nil {
			panic(err)
		}

		log.Infof("fetched %v users for %v, offset: %v", len(users.GroupUsers), login, offset)

		// Process users or groups in the response
		for _, groupUser := range users.GroupUsers {
			var document common.Document
			var idPrefix, docType string
			var metadata util.MapStr

			if groupUser.User != nil && cfg.IndexingUsers {
				idPrefix, docType = "yuque-user", groupUser.User.Type
				metadata = util.MapStr{
					"user_id":            groupUser.User.ID,
					"user_login":         groupUser.User.Login,
					"public":             groupUser.User.Public,
					"books_count":        groupUser.User.BooksCount,
					"follower_count":     groupUser.User.FollowersCount,
					"following_count":    groupUser.User.FollowingCount,
					"public_books_count": groupUser.User.PublicBooksCount,
				}

				document = common.Document{
					Source: common.DataSourceReference{
						//ID: "",//TODO
						Name: YuqueKey,
						Type: "connector",
					},
					Title:     groupUser.User.Name,
					Summary:   groupUser.User.Description,
					Type:      docType,
					URL:       fmt.Sprintf("https://infini.yuque.com/%v", groupUser.User.Login),
					Icon:      groupUser.User.AvatarURL, //TODO save to local store
					Thumbnail: groupUser.User.AvatarURL,
				}
				document.Created = &groupUser.User.CreatedAt
				document.Updated = &groupUser.User.UpdatedAt
				document.Metadata = metadata

			} else if groupUser.Group != nil && cfg.IndexingGroups {
				idPrefix, docType = "yuque-group", groupUser.Group.Type
				metadata = util.MapStr{
					"user_id":            groupUser.Group.ID,
					"user_login":         groupUser.Group.Login,
					"public":             groupUser.Group.Public,
					"books_count":        groupUser.Group.BooksCount,
					"member_count":       groupUser.Group.MembersCount,
					"public_books_count": groupUser.Group.PublicBooksCount,
				}

				document = common.Document{
					Source: common.DataSourceReference{
						//ID: "",//TODO
						Name: YuqueKey,
						Type: "connector",
					},
					Title:     groupUser.Group.Name,
					Summary:   groupUser.Group.Description,
					Type:      docType,
					URL:       fmt.Sprintf("https://infini.yuque.com/%v", groupUser.Group.ID),
					Icon:      groupUser.Group.AvatarURL, //TODO save to local store
					Thumbnail: groupUser.Group.AvatarURL,
				}
				document.Created = &groupUser.Group.CreatedAt
				document.Updated = &groupUser.Group.UpdatedAt
				document.Metadata = metadata
			}

			// Generate document ID and save
			if document.Title != "" {
				document.ID = util.MD5digest(fmt.Sprintf("%v-%v-%v", "test", idPrefix, metadata["user_id"]))
				log.Debugf("indexing user: %v, %v, %v", document.ID, metadata["user_login"], document.Title)
				this.save(document)
			}
		}

		// Exit loop if no more pages
		if len(users.GroupUsers) < pageSize || global.ShuttingDown() {
			break
		}
		offset += pageSize
	}

}
