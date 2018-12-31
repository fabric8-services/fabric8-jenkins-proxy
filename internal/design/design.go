package design // The convention consists of naming the design
// package "design"
import (
	. "github.com/goadesign/goa/design" // Use . imports to enable the DSL
	. "github.com/goadesign/goa/design/apidsl"
)

var _ = API("fabric8-jenkins-proxy-api", func() { // API defines the microservice endpoint and
	Title("Fabric8 Jenkins Proxy API") // other global properties. There should be one
	Scheme("http")                     // the design.
	Host("openshift.io")
	Trait("jsonapi-media-type", func() {
		ContentType("application/vnd.api+json")
	})
})

var _ = Resource("stats", func() { // Resources group related API endpoints
	BasePath("/api")    // together. They map to REST resources for REST
	DefaultMedia(Stats) // services.

	Action("info", func() { // Actions define a single API endpoint together
		Description("Get info by namespace") // with its path, parameters (both path
		Routing(GET("/info/:namespace"))     // parameters and querystring values) and payload
		Params(func() {                      // (shape of the request body).
			Param("namespace", String, "Namespace")
		})
		Response(OK)       // Responses define the shape and status code
		Response(NotFound) // of HTTP responses.
		Response(BadRequest, ErrorMedia)
	})
	Action("clear", func() { // Actions define a single API endpoint together
		Description("Get info by namespace") // with its path, parameters (both path
		Routing(DELETE("/clear/:namespace")) // parameters and querystring values) and payload
		Params(func() {                      // (shape of the request body).
			Param("namespace", String, "Namespace")
		})
		Response(OK)
		Response(BadRequest, ErrorMedia)
		Response(InternalServerError, ErrorMedia)
		Response(Unauthorized, ErrorMedia)
	})
})

// Stats defines the media type used to render response.
var Stats = MediaType("application/vnd.stats+json", func() {
	Description("Response from Fabric8-Jenkins-Proxy")
	Attributes(func() { // Attributes define the media type shape.
		Attribute("namespace", String, "Unique Namespace")
		Attribute("requests", Integer)
		Attribute("last_visit", DateTime)
		Attribute("last_request", DateTime)
		Required("namespace", "requests", "last_visit", "last_request")
	})
	View("default", func() { // View defines a rendering of the media type.
		Attribute("namespace") // Media types may have multiple views and must
		Attribute("requests")  // have a "default" view.
		Attribute("last_visit")
		Attribute("last_request")
	})
})
