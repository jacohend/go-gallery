package emails

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/gin-gonic/gin"
	"github.com/mikeydub/go-gallery/db/gen/coredb"
	"github.com/mikeydub/go-gallery/graphql/dataloader"
	"github.com/mikeydub/go-gallery/service/auth"
	"github.com/mikeydub/go-gallery/service/logger"
	"github.com/mikeydub/go-gallery/service/persist"
	"github.com/mikeydub/go-gallery/util"
	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

const emailsAtATime = 10_000

type VerificationEmailInput struct {
	UserID persist.DBID `json:"user_id" binding:"required"`
}

type verificationEmailTemplateData struct {
	Username string
	JWT      string
}

type errNoEmailSet struct {
	userID persist.DBID
}

func sendVerificationEmail(dataloaders *dataloader.Loaders, queries *coredb.Queries, s *sendgrid.Client) gin.HandlerFunc {

	return func(c *gin.Context) {
		var input VerificationEmailInput
		err := c.ShouldBindJSON(&input)
		if err != nil {
			util.ErrResponse(c, http.StatusBadRequest, err)
			return
		}

		user, err := dataloaders.UserByUserID.Load(input.UserID)
		if err != nil {
			util.ErrResponse(c, http.StatusBadRequest, err)
			return
		}

		if user.Email == "" {
			util.ErrResponse(c, http.StatusBadRequest, errNoEmailSet{userID: input.UserID})
			return
		}

		j, err := auth.JWTGeneratePipeline(c, input.UserID)
		if err != nil {
			util.ErrResponse(c, http.StatusBadRequest, err)
			return
		}

		logger.For(c).Debugf("sending verification email to %s with token %s", user.Email, j)

		from := mail.NewEmail("Gallery", viper.GetString("FROM_EMAIL"))
		to := mail.NewEmail(user.Username.String, user.Email.String())
		m := mail.NewV3Mail()
		m.SetFrom(from)
		p := mail.NewPersonalization()
		m.SetTemplateID(viper.GetString("SENDGRID_VERIFICATION_TEMPLATE_ID"))
		p.DynamicTemplateData = map[string]interface{}{
			"username":          user.Username.String,
			"verificationToken": j,
		}
		m.AddPersonalizations(p)
		p.AddTos(to)

		response, err := s.Send(m)
		if err != nil {
			util.ErrResponse(c, http.StatusInternalServerError, err)
			return
		}
		logger.For(c).Debugf("email sent: %+v", *response)

		c.Status(http.StatusOK)
	}
}

type notificationEmailDynamicTemplateData struct {
	Actor          string       `json:"actor"`
	Action         string       `json:"action"`
	CollectionName string       `json:"collectionName"`
	CollectionID   persist.DBID `json:"collectionId"`
	PreviewText    string       `json:"previewText"`
}
type notificationsEmailDynamicTemplateData struct {
	Notifications    []notificationEmailDynamicTemplateData `json:"notifications"`
	Username         string                                 `json:"username"`
	UnsubscribeToken string                                 `json:"unsubscribeToken"`
}

func sendNotificationEmails(queries *coredb.Queries, s *sendgrid.Client) gin.HandlerFunc {

	return func(c *gin.Context) {
		err := sendNotificationEmailsToAllUsers(c, queries, s)
		if err != nil {
			util.ErrResponse(c, http.StatusInternalServerError, err)
			return
		}

		c.Status(http.StatusOK)
	}
}

func autoSendNotificationEmails(queries *coredb.Queries, s *sendgrid.Client, psub *pubsub.Client) error {
	sub := psub.Subscription(viper.GetString("PUBSUB_NOTIFICATIONS_EMAILS_SUBSCRIPTION"))

	ctx := context.Background()
	return sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		err := sendNotificationEmailsToAllUsers(ctx, queries, s)
		if err != nil {
			logger.For(ctx).Errorf("error sending notification emails: %s", err)
			msg.Nack()
			return
		}
		msg.Ack()
	})
}

func sendNotificationEmailsToAllUsers(c context.Context, queries *coredb.Queries, s *sendgrid.Client) error {
	searchLimit := int32(10)
	resultLimit := 5
	return runForUsersWithNotificationsOnForEmailType(c, persist.EmailTypeNotifications, queries, func(u coredb.User) error {

		from := mail.NewEmail("Gallery", viper.GetString("FROM_EMAIL"))
		to := mail.NewEmail(u.Username.String, u.Email.String())
		m := mail.NewV3Mail()
		m.SetFrom(from)
		p := mail.NewPersonalization()
		m.SetTemplateID(viper.GetString("SENDGRID_NOTIFICATIONS_TEMPLATE_ID"))
		notifs, err := queries.GetRecentUnseenNotifications(c, coredb.GetRecentUnseenNotificationsParams{
			OwnerID: u.ID,
			Limit:   searchLimit,
		})
		if err != nil {
			return fmt.Errorf("failed to get notifications for user %s: %w", u.ID, err)
		}

		j, err := auth.JWTGeneratePipeline(c, u.ID)
		if err != nil {
			return fmt.Errorf("failed to generate jwt for user %s: %w", u.ID, err)
		}

		data := notificationsEmailDynamicTemplateData{
			Notifications:    make([]notificationEmailDynamicTemplateData, 0, resultLimit),
			Username:         u.Username.String,
			UnsubscribeToken: j,
		}
		notifTemplates := make(chan notificationEmailDynamicTemplateData)
		errChan := make(chan error)

		for _, n := range notifs {
			notif := n
			go func() {
				notifTemplate, err := notifToTemplateData(c, queries, notif)
				if err != nil {
					errChan <- err
					return
				}
				notifTemplates <- notifTemplate
			}()
		}

	outer:
		for i := 0; i < len(notifs); i++ {
			select {
			case err := <-errChan:
				logger.For(c).Errorf("failed to get notification template data: %v", err)
			case notifTemplate := <-notifTemplates:
				data.Notifications = append(data.Notifications, notifTemplate)
				if len(data.Notifications) >= resultLimit {
					break outer
				}
			}
		}

		if len(data.Notifications) == 0 {
			return nil
		}

		asJSON, err := json.Marshal(data)
		if err != nil {
			return err
		}
		asMap := make(map[string]interface{})
		err = json.Unmarshal(asJSON, &asMap)
		if err != nil {
			return err
		}

		logger.For(c).Debugf("sending notifications email to %s with data %+v", u.Email, asMap)

		p.DynamicTemplateData = asMap

		m.Asm = &mail.Asm{GroupID: viper.GetInt("SENDGRID_UNSUBSCRIBE_NOTIFICATIONS_GROUP_ID")}
		m.AddPersonalizations(p)
		p.AddTos(to)

		response, err := s.Send(m)
		if err != nil {
			return err
		}
		logger.For(c).Debugf("email sent: %d", *&response.StatusCode)
		return nil
	})
}

func notifToTemplateData(ctx context.Context, queries *coredb.Queries, n coredb.Notification) (notificationEmailDynamicTemplateData, error) {

	switch n.Action {
	case persist.ActionAdmiredFeedEvent:
		feedEvent, err := queries.GetFeedEventByID(ctx, n.FeedEventID)
		if err != nil {
			return notificationEmailDynamicTemplateData{}, fmt.Errorf("failed to get feed event for admire %s: %w", n.FeedEventID, err)
		}
		collection, _ := queries.GetCollectionById(ctx, feedEvent.Data.CollectionID)
		data := notificationEmailDynamicTemplateData{}
		if collection.ID != "" && collection.Name.String != "" {
			data.CollectionID = collection.ID
			data.CollectionName = collection.Name.String
			data.Action = "admired your additions to"
		} else {
			data.Action = "admired your gallery update"
		}
		if len(n.Data.AdmirerIDs) > 1 {
			data.Actor = fmt.Sprintf("%d collectors", len(n.Data.AdmirerIDs))
		} else {
			actorUser, err := queries.GetUserById(ctx, n.Data.AdmirerIDs[0])
			if err != nil {
				return notificationEmailDynamicTemplateData{}, err
			}
			data.Actor = actorUser.Username.String
		}
		return data, nil
	case persist.ActionUserFollowedUsers:
		if len(n.Data.FollowerIDs) > 1 {
			return notificationEmailDynamicTemplateData{
				Actor:  fmt.Sprintf("%d users", len(n.Data.FollowerIDs)),
				Action: "followed you",
			}, nil
		}
		if len(n.Data.FollowerIDs) == 1 {
			userActor, err := queries.GetUserById(ctx, n.Data.FollowerIDs[0])
			if err != nil {
				return notificationEmailDynamicTemplateData{}, fmt.Errorf("failed to get user for follower %s: %w", n.Data.FollowerIDs[0], err)
			}
			action := "followed you"
			if n.Data.FollowedBack {
				action = "followed you back"
			}
			return notificationEmailDynamicTemplateData{
				Actor:  userActor.Username.String,
				Action: action,
			}, nil
		}
		return notificationEmailDynamicTemplateData{}, fmt.Errorf("no follower ids")
	case persist.ActionCommentedOnFeedEvent:
		comment, err := queries.GetCommentByCommentID(ctx, n.CommentID)
		if err != nil {
			return notificationEmailDynamicTemplateData{}, fmt.Errorf("failed to get comment for comment %s: %w", n.CommentID, err)
		}
		userActor, err := queries.GetUserById(ctx, comment.ActorID)
		if err != nil {
			return notificationEmailDynamicTemplateData{}, fmt.Errorf("failed to get user for comment actor %s: %w", comment.ActorID, err)
		}
		feedEvent, err := queries.GetFeedEventByID(ctx, n.FeedEventID)
		if err != nil {
			return notificationEmailDynamicTemplateData{}, fmt.Errorf("failed to get feed event for comment %s: %w", n.FeedEventID, err)
		}
		collection, _ := queries.GetCollectionById(ctx, feedEvent.Data.CollectionID)
		if collection.ID != "" {
			return notificationEmailDynamicTemplateData{
				Actor:          userActor.Username.String,
				Action:         "commented on your additions to",
				CollectionName: collection.Name.String,
				CollectionID:   collection.ID,
				PreviewText:    util.TruncateWithEllipsis(comment.Comment, 20),
			}, nil
		}
		return notificationEmailDynamicTemplateData{
			Actor:       userActor.Username.String,
			Action:      "commented on your gallery update",
			PreviewText: util.TruncateWithEllipsis(comment.Comment, 20),
		}, nil
	case persist.ActionViewedGallery:
		if len(n.Data.AuthedViewerIDs)+len(n.Data.UnauthedViewerIDs) > 1 {
			return notificationEmailDynamicTemplateData{
				Actor:  fmt.Sprintf("%d collectors", len(n.Data.AuthedViewerIDs)+len(n.Data.UnauthedViewerIDs)),
				Action: "viewed your gallery",
			}, nil
		}
		if len(n.Data.AuthedViewerIDs) == 1 {
			userActor, err := queries.GetUserById(ctx, n.Data.AuthedViewerIDs[0])
			if err != nil {
				return notificationEmailDynamicTemplateData{}, fmt.Errorf("failed to get user for viewer %s: %w", n.Data.AuthedViewerIDs[0], err)
			}
			return notificationEmailDynamicTemplateData{
				Actor:  userActor.Username.String,
				Action: "viewed your gallery",
			}, nil
		}
		if len(n.Data.UnauthedViewerIDs) == 1 {
			return notificationEmailDynamicTemplateData{
				Actor:  "Someone",
				Action: "viewed your gallery",
			}, nil
		}

		return notificationEmailDynamicTemplateData{}, fmt.Errorf("no viewer ids")
	default:
		return notificationEmailDynamicTemplateData{}, fmt.Errorf("unknown action %s", n.Action)
	}
}

func runForUsersWithNotificationsOnForEmailType(ctx context.Context, emailType persist.EmailType, queries *coredb.Queries, fn func(u coredb.User) error) error {
	errGroup := new(errgroup.Group)
	var lastID persist.DBID
	var lastCreatedAt time.Time
	var endTime time.Time = time.Now().Add(24 * time.Hour)
	requiredStatus := persist.EmailVerificationStatusVerified
	if isDevEnv() {
		requiredStatus = persist.EmailVerificationStatusAdmin
	}
	for {
		users, err := queries.GetUsersWithEmailNotificationsOnForEmailType(ctx, coredb.GetUsersWithEmailNotificationsOnForEmailTypeParams{
			Limit:         emailsAtATime,
			CurAfterTime:  lastCreatedAt,
			CurBeforeTime: endTime,
			CurAfterID:    lastID,
			PagingForward: true,
			EmailVerified: requiredStatus,
			Column1:       emailType.String(),
		})
		if err != nil {
			return err
		}

		for _, user := range users {
			u := user
			errGroup.Go(func() error {
				err = fn(u)
				if err != nil {
					return err
				}
				return nil
			})
		}

		if len(users) < emailsAtATime {
			break
		}

		if len(users) > 0 {
			lastUser := users[len(users)-1]
			lastID = lastUser.ID
			lastCreatedAt = lastUser.CreatedAt
		}
	}

	return errGroup.Wait()
}

func (e errNoEmailSet) Error() string {
	return fmt.Sprintf("user %s has no email", e.userID)
}
