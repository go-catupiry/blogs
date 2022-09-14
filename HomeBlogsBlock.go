package blog

import (
	"github.com/go-catupiry/catu"
	"github.com/sirupsen/logrus"
)

type HomeBlogsBlockTPL struct {
	Ctx            *catu.RequestContext
	Record         interface{}
	HasLatestPosts *bool
	LatestPosts    *[]*BlogPostModel
}

func LoadHomeBlogPostBlockData(ctx *catu.RequestContext) error {
	var err error
	ctx.MetaTags.Title = "Últimas Postagens no Sertão Carioca"

	var latestPosts []*BlogPostModel
	err = BlogPostFindLatest(&latestPosts, 4)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"err": err,
		}).Error("LoadHomeBlogsBlockData error on find latest posts")
		return err
	}

	var teaserList []string

	logrus.WithFields(logrus.Fields{
		"len_records_found": len(latestPosts),
	}).Debug("BlogPostFindAll count result")

	for i := range latestPosts {
		latestPosts[i].LoadTeaserData()

		teaserHTML, err := latestPosts[i].RenderHomeBlogPostBlock(ctx)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"err": err.Error(),
			}).Error("HomeBlogsPostBlock error on render block")
		} else {
			teaserList = append(teaserList, teaserHTML.String())
		}
	}

	ctx.Set("hasBlogPostRecords", true)
	ctx.Set("blogPostRecords", teaserList)

	return err
}
