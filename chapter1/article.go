package main

import (
	"flag"
	"fmt"
	"redis-in-action-by-go/cache"
	"strconv"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
)

type Article struct {
	Title  string `redis:"title"`
	Link   string `redis:"link"`
	Time   int64  `redis:"time"`
	Poster string `redis:"poster"`
	Votes  int    `redis:"votes"`
}

const (
	OneWeekInSeconds = 7 * 86400
	VoteScore        = 432
	ArticlePerPage   = 25
)

func main() {
	flag.Parse()

	cache.InitRedis()

	_ = Article{
		Title:  "title",
		Link:   "link",
		Time:   time.Now().Unix(),
		Poster: "poster",
		Votes:  1,
	}
	//PostArticle(&article)

	getGroupArticle("group:test", 1, "time:")
}

func PostArticle(article *Article) {

	conn := cache.GetRedisConn()
	defer conn.Close()

	reply, err := redis.Int(conn.Do("incr", "article:"))
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(reply)
	//replyInt, ok := reply.(int)

	articleID := strconv.Itoa(reply)
	voted := "voted:" + articleID

	// 将作者加入到已赞读者set中
	_, err = conn.Do("sadd", voted, article.Poster)
	checkErr(err)

	nowUnix := time.Now().Unix()

	// 将文章放入按照时间和分数排序的zset中
	articleIndex := "article:" + articleID
	_, err = conn.Do("zadd", "score:", nowUnix+VoteScore, articleIndex)
	checkErr(err)

	_, err = conn.Do("zadd", "time:", nowUnix, articleIndex)
	checkErr(err)

	// 将文章放在文章的hash

	_, err = conn.Do("hmset", redis.Args{}.Add(articleIndex).AddFlat(article)...)

	/*

		if _, err := c.Do("HMSET", redis.Args{}.Add("id1").AddFlat(&p1)...); err != nil {
		    fmt.Println(err)
		    return
		}*/
	checkErr(err)

	fmt.Printf("list: voted = %s article = %s\n", voted, articleIndex)

}

func articleVote(user, articleIndex string) {
	conn := cache.GetRedisConn()
	defer conn.Close()

	lastWeekUnix := time.Now().Unix() - OneWeekInSeconds

	reply, err := redis.Int64(conn.Do("zscore", "time:", articleIndex))
	checkErr(err)

	// 如果在一周内，增加人，
	if reply > lastWeekUnix {
		articleID := strings.Split(articleIndex, ":")[1]
		replyInt, err := redis.Int64(conn.Do("sadd", "voted:"+articleID, user))
		checkErr(err)

		// 插入成功，增加分数，增加文章投票数
		if replyInt != 0 {

			_, err = conn.Do("zincrby", "score:", VoteScore)
			checkErr(err)

			_, err = conn.Do("hincrby", articleIndex, "votes", 1)
			checkErr(err)
		}
	}

}

func getArticles(page int, order string) map[string]map[string]string {

	conn := cache.GetRedisConn()
	defer conn.Close()

	start := (page - 1) * ArticlePerPage
	end := start + ArticlePerPage - 1

	articleIndexs, err := redis.Strings(conn.Do("revrange", start, end))
	checkErr(err)

	articles := make(map[string]map[string]string, 0)
	for _, articleIndex := range articleIndexs {
		reply, err := redis.StringMap(conn.Do("hgetall", articleIndex))
		checkErr(err)
		articles[articleIndex] = reply

	}
	fmt.Println(articles)
	return articles

}

func addRemoveGroups(articleID int, toAdds, toRemoves []string) {

	conn := cache.GetRedisConn()
	defer conn.Close()

	articleIndex := "article:" + strconv.Itoa(articleID)

	for _, group := range toAdds {
		_, err := conn.Do("sadd", "group:"+group, articleIndex)
		checkErr(err)
	}

	for _, group := range toRemoves {
		_, err := conn.Do("srem", "group:"+group, articleIndex)
		checkErr(err)
	}

}

func getGroupArticle(group string, page int, order string) {
	conn := cache.GetRedisConn()
	defer conn.Close()

	key := order + group

	isExists, err := redis.Bool(conn.Do("exists", key))
	checkErr(err)

	if !isExists {
		_, err := conn.Do("zinterstore", key, 2, "group:"+group, order, "WEIGHTS", 2, 3, "aggregate", "max")
		fmt.Println("aa")
		checkErr(err)
		fmt.Println("aa")
	}
}

func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
