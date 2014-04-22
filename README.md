# Golang router

Router is a standalone router used in the Shadow framework. It's goal is to be a flexible router that is fast and handles validation so your controllers don't have to. It also handles extensions nicely so you don't need to define them with regexes in every route.

Note router is still a work-in-progress. Things will change - please don't use in production yet.

Router borrows the concept from @nikic's FastRoute to speed up route matching. Routes are grouped into a single regex of 15 at a time to identify matches. Learn more here: http://nikic.github.io/2014/02/18/Fast-request-routing-using-regular-expressions.html

Here is the breakdown of sample route definitions to max regexes evaluated

```
10 routes = 1 regex checked
30 routes = Max 2 regexes checked
46 routes = Max 4 regexes checked
```

Routes are grouped by HTTP method, so if you have 75 GET routes but only 4 POST routes, a POST request will only require a max of 1 regex to be evaluated - not 5 - to attempt to find a match.

## Instantiating your router

Use the New() method to instantiate your router

~~~ go
router := router.New()
~~~

## Named Parameters

All routes require a name and a path. Named parameters are defined using :name syntax. Extensions are always removed before Router matches paths (see "Defining Valid Extensions"). Examples:

~~~ go
router.Get("articles_read", "/articles/:slug")
router.Get("articles_read", "/articles/:channel/:id/:slug")

// Post routes
router.Post("articles_create", "/articles")

// Put routes
router.Put("articles_update", "/articles/:id")

// Delete routes
router.Delete("articles_delete", "/articles/:id")
~~~

## Regex Validation

Regex validation can be used on named parameters by providing a regex in parentheses following the param name

~~~go
router.Get("articles_read", "/articles/:slug([a-z]+)")
router.Put("articles_update", "/articles/:id([0-9]+)")
~~~

You can also use the following convenience methods which are converted to regexes once (rather than on every request)

~~~go
// id should be an integer [0-9]+
router.Get("articles_read", "/articles/:id(int)")

// name should be alpha only [a-z]
router.Get("company_profile", "/company/:name(alpha)")

// slug should be [a-z0-9-]+
router.Get("articles_read", "/articles/:id/:slug(slug)")

// name should be alphanumeric only [a-z0-9]
router.Get("company_profile", "/company/:name(alphanumeric)")

// token should be an md5 hash [0-9a-fA-F]{32}
router.Get("password_reset", "/users/reset_password/:token(md5)")

// id should be a mongo id [0-9a-fA-F]{24}
router.Get("articles_read", "/articles/:id(mongo)")
~~~

## Generating URLs

You can generate urls with either the Router struct or a Route struct

~~~go
// From the Router struct: requires the route name
url := router.URL("articles_read", "channel", "entertainment", "id", 10, "slug", "a-b-c")

// Or with the Router struct directly
url := route.URL("channel", "entertainment", "id", 10, "slug", "a-b-c")
~~~

## Defining Valid Extensions

Router allows you to specify what extensions are valid. Any path you specify is never matched against the extension directly, so you have to specify which extensions are valid.

~~~go
router := New()
router.ValidExtensions("", "json", "csv")
router.Get("read", "/articles/:id(int)")
~~~

An empty string represents no extension (and is the default). Given the above definition, the following routes are valid:

```
GET /articles/10
```

```
GET /articles/10.json
```

```
GET /articles/10.csv
```

However this is not:

```
GET /articles/10.xml
```

If ValidExtensions() was only set to allow "json" and "csv", one of the two extensions would be mandatory. /articles/10 would no longer match (because no extensions - or "" - is no longer allowed).

## Compiling the router

Because Router doesn't loop through every individual route to find a match, the routes need to be compiled before you can look for a match. You do this by calling the compile method once like this:

~~~go
router.Compile()
~~~

## Dispatching your router

Once you have compiled your routes, you can call the Dispatch() method. The dispatch method returns the Route struct and params.

~~~go
// Given this definition:
router.ValidExtensions("", "json")
router.Get("articles_read", "/articles/:id(int)")

// This will match the "articles_read" route
route, params := router.Dispatch("GET", "/articles/10.json")
~~~

Params will contain two parameters:

ext = "json"
id = 10

The extension will always be sent in the params - even if empty (no extension).

Any numeric params (i.e. 10) will always be an integer rather than a string.

## Route Groups

Route groups just provide convenience and a nice syntax for defining groups of routes. Given the following:

~~~go
router.Group("/articles", "articles", func(r *Router) {
    r.Get("read", "/:id(int)")
    r.Get("section", "/:section(slug)/:id(int)")
    r.Get("mongo", "/:id(mongo)")
})
~~~

Is identical to this:

~~~go
Router.Get("articles_read", "/articles/:id(int)")
Router.Get("articles_section", "/articles/:section(slug)/:id(int)")
Router.Get("articles_mongo", "/articles/:id(mongo)")
~~~

Note the name of the route is prefixed with articles_ and the path is prefixed with /articles - this is essentially all Group() does. The method signature is likely to change so that the name is defined first in the Group().