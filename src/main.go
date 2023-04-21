package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/tidwall/gjson"
)

const reverseProxyUrl = "https://pimg.rem.asia"

func saveFile(content []byte, savePath string, fileNmae string) error {

	output, err := os.Create(savePath + fileNmae) // create a file to save file. say..pic.jpg. 				// save. 				// name
	if err != nil {
		fmt.Println("create file failed!", err) // name of the file. 				// save. 				// name.)
		return err
	}
	defer output.Close()
	_, err = io.Copy(output, bytes.NewReader(content)) // save the file. say. 			// name of the file.

	if err != nil {
		fmt.Println("copy failed!", err) // name of the file. 			// save. 				// name of the file.
		return err
	}

	fmt.Println(fileNmae + " downloaded")

	return nil
}

// 根据插画ID搜索并下载
func searchByIllustId(id string) error {
	baseURL := "https://www.pixiv.net/touch/ajax/illust/details?illust_id=" + id
	savePath := "./illusts_" + id + "/" // path where the image will be saved. say./pixiv_images/

	c := colly.NewCollector(colly.Async(true))

	c.Limit(&colly.LimitRule{
		Parallelism: runtime.NumCPU(),
		RandomDelay: 5 * time.Second,
	})

	c.WithTransport(&http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   120 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	})

	imgC := c.Clone()

	imgC.Limit(&colly.LimitRule{
		Parallelism: runtime.NumCPU(),
		RandomDelay: 5 * time.Second,
	})

	imgC.OnError(func(r *colly.Response, err error) {
		fmt.Println("Download illust error", err)
	})

	imgC.OnResponse(func(resp *colly.Response) {
		path := strings.Split(resp.Request.URL.String(), "/")
		imgName := path[len(path)-1] // name of the image. say. "pixiv_id-date.jpg"
		saveFile(resp.Body, savePath, imgName)
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Get illust's detail error", err)
	})

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Add("referer", "https://www.pixiv.net/")
	})

	c.OnResponse(func(r *colly.Response) {
		res := gjson.Get(string(r.Body), "body.illust_details.manga_a")

		// create directory if not exists.
		if os.Mkdir(savePath, os.ModePerm) != nil {
			fmt.Println("Error: can't create directory") // debug only.
			savePath = "./"
		}

		if res.Raw == "" {
			link := gjson.Get(string(r.Body), "body.illust_details.url_big")
			parsed, _ := url.Parse(link.Str)
			imgC.Visit(reverseProxyUrl + parsed.Path)
		} else {
			res.ForEach(func(key, value gjson.Result) bool {
				link := gjson.Get(value.Raw, "url_big")
				parsed, _ := url.Parse(link.Str)
				imgC.Visit(reverseProxyUrl + parsed.Path)
				return true
			})
		}
	})

	c.Visit(baseURL) // Visit the URL for the desired page. 			// returns immediately. 			// Visit does not
	c.Wait()
	imgC.Wait()

	return nil
}

// 根据画师ID搜索并下载该画师的所有画品
func searchByUid(uid string) error {
	baseURL := "https://www.pixiv.net/ajax/user/" + uid + "/profile/all"

	c := colly.NewCollector()
	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Get user's detail error", err)
	})
	c.OnResponse(func(r *colly.Response) {
		imgs := gjson.Get(string(r.Body), "body.illusts")
		imgs.ForEach(func(key, value gjson.Result) bool {
			searchByIllustId(key.Str)
			return true
		})
	})

	c.Visit(baseURL)
	return nil
}

func menu() {
	fmt.Println("1.Dowanload image by illust's id")
	fmt.Println("2.Dowanload image by uid")
	fmt.Println("3.exit")
	fmt.Println("input your option")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	switch scanner.Text() {
	case "1":
		fmt.Println("Please enter the illust's id of the image you want to download: ")
		scanner.Scan()
		searchByIllustId(scanner.Text())
	case "2":
		fmt.Println("Please enter the uid of the image you want to download: ")
		scanner.Scan()
		searchByUid(scanner.Text())
	default:
		fmt.Println("Goodbye!")
	}
}

func main() {
	menu()
}
