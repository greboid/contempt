package sources

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	tt "text/template"

	"github.com/csmith/contempt/pkg/template"
)

func HttpSource() template.FunctionSource {
	return func(writer template.BomWriter) tt.FuncMap {
		return tt.FuncMap{
			"regex_url_content": func(name, url, regex string) (string, error) {
				res, err := regexURLContent(url, regex)
				if err != nil {
					return "", err
				}
				res = writer.Write(fmt.Sprintf("regexurl:%s", name), res)
				return res, nil
			},
		}
	}
}

func regexURLContent(url string, regex string) (string, error) {
	re, err := regexp.Compile(regex)
	if err != nil {
		return "", err
	}

	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	result := re.FindSubmatch(body)

	if len(result) == 0 {
		return "", errors.New("no match found")
	}
	return string(result[1]), nil
}
