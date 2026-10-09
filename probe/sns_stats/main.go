// sns_stats 统计指定发布者的朋友圈内容：动态数、媒体项数、时间范围、评论/点赞数。
package main

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"time"

	_ "modernc.org/sqlite"
)

func main() {
	dbPath := `F:\wxff\data\backup\_plain\sns__sns.db`
	user := "wxid_i49id3jztuzq12"
	if len(os.Args) > 1 {
		user = os.Args[1]
	}
	db, _ := sql.Open("sqlite", dbPath)
	defer db.Close()

	rows, err := db.Query("SELECT content FROM SnsTimeLine WHERE user_name=? ORDER BY tid ASC", user)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	mediaRe := regexp.MustCompile(`<media>`)
	timeRe := regexp.MustCompile(`<createTime>(\d+)</createTime>`)
	descRe := regexp.MustCompile(`<contentDesc>([^<]*)</contentDesc>`)
	commentRe := regexp.MustCompile(`<user_comment>`)
	likeRe := regexp.MustCompile(`<like_user_list>`)
	videoRe := regexp.MustCompile(`<type>6</type>`)

	posts := 0
	mediaTotal := 0
	comments := 0
	likes := 0
	videos := 0
	withText := 0
	minT, maxT := int64(1<<62), int64(0)
	for rows.Next() {
		var c string
		rows.Scan(&c)
		posts++
		mediaTotal += len(mediaRe.FindAllString(c, -1))
		comments += len(commentRe.FindAllString(c, -1))
		likes += len(likeRe.FindAllString(c, -1))
		videos += len(videoRe.FindAllString(c, -1))
		if m := descRe.FindStringSubmatch(c); len(m) > 1 && m[1] != "" {
			withText++
		}
		if m := timeRe.FindStringSubmatch(c); len(m) > 1 {
			var t int64
			fmt.Sscanf(m[1], "%d", &t)
			if t < minT {
				minT = t
			}
			if t > maxT {
				maxT = t
			}
		}
	}
	fmt.Printf("发布者: %s\n", user)
	fmt.Printf("动态总数: %d\n", posts)
	fmt.Printf("含文字: %d\n", withText)
	fmt.Printf("媒体项总数: %d\n", mediaTotal)
	fmt.Printf("  其中视频(type=6): %d\n", videos)
	fmt.Printf("评论数: %d\n", comments)
	fmt.Printf("点赞数: %d\n", likes)
	if maxT > 0 {
		fmt.Printf("时间范围: %s ~ %s\n",
			time.Unix(minT, 0).Format("2006-01-02"),
			time.Unix(maxT, 0).Format("2006-01-02"))
	}
}
