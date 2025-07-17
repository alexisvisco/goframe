package httpx

import "net/http"

type RedirectResponse struct {
	url        string
	statusCode int
}

func (r RedirectResponse) WriteTo(w http.ResponseWriter, req *http.Request) error {
	http.Redirect(w, req, r.url, r.statusCode)
	return nil
}

func NewRedirectResponse(statusCode int, url string) RedirectResponse {
	allowedStatusCodes := map[int]bool{
		http.StatusMovedPermanently:  true,
		http.StatusFound:             true,
		http.StatusSeeOther:          true,
		http.StatusTemporaryRedirect: true,
		http.StatusPermanentRedirect: true,
	}
	if _, ok := allowedStatusCodes[statusCode]; !ok {
		// If the status code is not one of the allowed redirect codes, default to 302 Found
		statusCode = http.StatusFound
	}
	return RedirectResponse{
		url:        url,
		statusCode: statusCode,
	}
}
