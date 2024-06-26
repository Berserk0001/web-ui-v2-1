package action

import (
	"context"
	"net/http"
	"time"

	m "github.com/webtor-io/web-ui-v2/services/models"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	api "github.com/webtor-io/web-ui-v2/services/api"
	"github.com/webtor-io/web-ui-v2/services/job"
	"github.com/webtor-io/web-ui-v2/services/template"
	wj "github.com/webtor-io/web-ui-v2/services/web/job"
	"github.com/webtor-io/web-ui-v2/services/web/job/script"
)

type PostArgs struct {
	ResourceID string
	ItemID     string
	Claims     *api.Claims
}

type TrackPutArgs struct {
	ID         string `json:"id"`
	ResourceID string `json:"resourceID"`
	ItemID     string `json:"itemID"`
}

type PostData struct {
	Job  *job.Job
	Args *PostArgs
}

type Handler struct {
	jobs *wj.Handler
	tb   template.Builder
}

func RegisterHandler(c *cli.Context, r *gin.Engine, tm *template.Manager, jobs *wj.Handler) {
	h := &Handler{
		tb:   tm.MustRegisterViews("action/*").WithHelper(NewHelper()),
		jobs: jobs,
	}
	r.POST("/download-file", func(c *gin.Context) {
		h.post(c, "download")
	})
	r.POST("/download-dir", func(c *gin.Context) {
		h.post(c, "download")
	})
	r.POST("/preview-image", func(c *gin.Context) {
		h.post(c, "preview-image")
	})
	r.POST("/stream-audio", func(c *gin.Context) {
		h.post(c, "stream-audio")
	})
	r.POST("/stream-video", func(c *gin.Context) {
		h.post(c, "stream-video")
	})
	r.PUT("/stream-video/subtitle", func(c *gin.Context) {
		a := TrackPutArgs{}
		if err := c.BindJSON(&a); err != nil {
			c.Error(err)
			return
		}
		vsud := m.NewVideoStreamUserData(a.ResourceID, a.ItemID)
		vsud.SubtitleID = a.ID
		if err := vsud.UpdateSessionData(c); err != nil {
			c.Error(err)
		}
	})
	r.PUT("/stream-video/audio", func(c *gin.Context) {
		a := TrackPutArgs{}
		if err := c.BindJSON(&a); err != nil {
			c.Error(err)
			return
		}
		vsud := m.NewVideoStreamUserData(a.ResourceID, a.ItemID)
		vsud.AudioID = a.ID
		if err := vsud.UpdateSessionData(c); err != nil {
			c.Error(err)
		}
	})
}

func (s *Handler) bindPostArgs(c *gin.Context) (*PostArgs, error) {
	rID, ok := c.GetPostFormArray("resource-id")
	if !ok {
		return nil, errors.Errorf("no resource id provided")
	}
	iID, ok := c.GetPostFormArray("item-id")
	if !ok {
		return nil, errors.Errorf("no item id provided")
	}

	return &PostArgs{
		ResourceID: rID[0],
		ItemID:     iID[0],
		Claims:     api.GetClaimsFromContext(c),
	}, nil
}

func (s *Handler) post(c *gin.Context, action string) {
	var (
		d    PostData
		err  error
		args *PostArgs
		job  *job.Job
	)
	postTpl := s.tb.Build("action/post")
	args, err = s.bindPostArgs(c)
	if err != nil {
		postTpl.HTMLWithErr(errors.Wrap(err, "wrong args provided"), http.StatusBadRequest, c, d)
		return
	}
	d.Args = args
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Minute)
	job, err = s.jobs.Action(ctx, c, args.Claims, args.ResourceID, args.ItemID, action, &script.StreamSettings{})
	if err != nil {
		postTpl.HTMLWithErr(errors.Wrap(err, "failed to start downloading"), http.StatusBadRequest, c, d)
		return
	}
	d.Job = job
	postTpl.HTML(http.StatusOK, c, d)
}
