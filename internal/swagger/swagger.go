package swagger

import (
	"fmt"
	"net/http"
)

const swaggerUIVersion = "5.18.2"

// Handler returns an HTTP handler that serves Swagger UI
func Handler(specEndpoint string) http.HandlerFunc {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@%s/swagger-ui.css" crossorigin="anonymous">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@%s/swagger-ui-bundle.js" crossorigin="anonymous"></script>
  <script>
    SwaggerUIBundle({
      url: '%s',
      dom_id: '#swagger-ui',
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIBundle.SwaggerUIStandalonePreset
      ],
      layout: "BaseLayout"
    });
  </script>
</body>
</html>`, swaggerUIVersion, swaggerUIVersion, specEndpoint)

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	}
}
