package main

import (
	"fmt"
)

type htmlData struct {
	head   string
	body   string
	script string
}

const (
	htmlModel = `
<html>
	<head> 
		<title>%v</title> 
		%v
	</head> 
	
	<body> 
		%v
	</body>
	
	</script>
		%v
	</script>
</html>
`
)

func getRenderHtml(title string, hd []htmlData) string {
	var (
		head   string
		body   string
		script string
	)
	enter := "\n"
	for _, v := range hd {
		head += v.head + enter
		body += v.body + enter
		script += v.script + enter
	}

	return fmt.Sprintf(htmlModel, title, head, body, script)
}
