/*
Copyright © 2024 Dr. Harald Leinders <harald@leinders.de>
*/
package cmd

import (
	"fmt"
	"net/http"
	"strings"

	at "github.com/hleinders/AnsiTerm"
	"github.com/spf13/cobra"
)

type CookieFlags struct {
	follow              bool
	displaySingleCookie []string
}

var cookieFlags CookieFlags

var cookieShortDesc = "Shows the request and response cookies of a http request"

// cookiesCmd represents the cookies command
var cookiesCmd = &cobra.Command{
	Use:     "cookies <URL> [<URL> ...]",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"ck", "cookie"},
	Short:   cookieShortDesc,
	Long: makeHeader(lowerAppName+" cookies: "+cookieShortDesc) + `With command 'cookies', all request and response cookies
are shown. You may pass the '-f|--follow' flag to follow redirects.
In this case, the cookies can be displayed in any hop with the '-a|--all' flag.

Flags marked with '***' may be used multiple times.`,
	Run: func(cmd *cobra.Command, args []string) {
		ExecCookies(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(cookiesCmd)

	// flags
	cookiesCmd.Flags().BoolVarP(&cookieFlags.follow, "follow", "f", false, "show all response cookies")

	// Parameter
	cookiesCmd.Flags().StringSliceVarP(&cookieFlags.displaySingleCookie, "show-cookie", "S", nil, "show only cookie `FOOBAR`; ***")

}

func ExecCookies(cmd *cobra.Command, args []string) {
	var hops []WebRequestResult
	var err error

	// create template request:
	req := WebRequest{
		agent:     rootFlags.agent,
		lang:      rootFlags.reqLang,
		method:    rootFlags.httpMethod,
		authUser:  rootFlags.authUser,
		authPass:  rootFlags.authPass,
		reqBody:   globalRequestBody,
		xhdrs:     globalHeaderList,
		cookieLst: globalCookieLst,
	}

	for _, rawURL := range args {
		newReq := req
		newReq.url, err = checkURL(rawURL, false)
		check(err, ErrNoURL)

		// handle the request(s)
		if cookieFlags.follow {
			hops, err = follow(&newReq, &connSet)
			if err != nil {
				pr.Error(err.Error())
			}
		} else {
			hops, err = noFollow(&newReq, &connSet)
			if err != nil {
				pr.Error(err.Error())
			}

		}

		// display results
		prettyPrintCookies(hops)
	}
}

func makeCookiesFromNames(cookieList []*http.Cookie, names []string) []*http.Cookie {
	var cl []*http.Cookie

	for _, c := range cookieList {
		cName := c.Name
		if found := findInSlice(names, cName); found {
			cl = append(cl, c)
		}
	}
	return cl
}

func makeCookiesFromResponse(headers http.Header) []*http.Cookie {
	var cl []*http.Cookie

	if cookieHeader := headers.Values("Set-Cookie"); len(cookieHeader) > 0 {
		c := http.Cookie{Name: "Set-Cookie", Value: "xxxx"}
		cl = append(cl, &c)
	}
	return cl
}

func chainPrintCookies(indent, frameChar, mark, titleMsg string, cookieList []*http.Cookie) {
	// var headerVal string
	// var cookieKeys []string

	fmtString := "%s%s   %s\n"
	fmt.Printf(fmtString, indent, frameChar, at.Bold(titleMsg))

	if len(cookieList) > 0 {
		for _, c := range cookieList {
			ckStr := fmt.Sprintf("%s: %s", c.Name, fullCookieValues(c))
			fmt.Printf(fmtString, indent, frameChar, fmt.Sprintf("%s %s", mark, shorten(rootFlags.long, screenWidth-25, ckStr)))
		}
	} else {
		fmt.Printf(fmtString, indent, frameChar, fmt.Sprintf("%s %s", mark, "(None)"))
	}
	fmt.Printf("%s%s\n", indent, frameChar)
}

func fullCookieValues(c *http.Cookie) string {
	fv := c.Value

	if c.Path != "" {
		fv = fmt.Sprintf("%s; path=%s", fv, c.Path)
	}

	if c.Domain != "" {
		fv = fmt.Sprintf("%s; domain=%s", fv, c.Domain)
	}

	return fv
}

// func fullCookieString(c *http.Cookie) string {
// 	return fmt.Sprintf("%s=%s", c.Name, fullCookieValues(c))
// }

func prettyPrintCookies(resultList []WebRequestResult) {

	fmt.Println()

	for cnt, h := range resultList {
		// result title
		title := fmt.Sprintf("%d:  %s (%s)", cnt+1, h.PrettyPrintRedir(cnt), colorStatus(h.response.StatusCode))
		titleLen := len(stripColorCodes(title))

		fmt.Println(title)
		fmt.Println(strings.Repeat(at.FrameOHLine, titleLen))
		fmt.Println()

		// print cookies
		ckHandleCookies(h)
		fmt.Println()
	}
}

func ckHandleCookies(result WebRequestResult) {
	if len(cookieFlags.displaySingleCookie) == 0 {
		// Request cookies from globalCookieList
		chainPrintCookies(indentHeader, "", at.BulletChar, "Request Cookies:", result.request.Cookies())

		// Set-Cookie in response headers?
		responseCookies := makeCookiesFromResponse(result.response.Header)

		if len(responseCookies) > 0 {
			chainPrintHeaders(indentHeader, "", at.BulletChar, at.Yellow("Cookie Store Request Detected:"), makeHeadersFromName([]string{"Set-Cookie"}, result.response.Header))
		}

		if result.cookieLst != nil {
			chainPrintCookies(indentHeader, "", at.BulletChar, "Stored Cookies:", result.cookieLst)
		}
	} else {
		chainPrintCookies(indentHeader, "", at.BulletChar, "Selected Cookies:", makeCookiesFromNames(result.cookieLst, cookieFlags.displaySingleCookie))
	}
}
