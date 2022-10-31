package graphqlutil

import (
	"context"
	"encoding/json"
	htmltemplate "html/template"
	"net/http"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
)

// ref: https://github.com/graphql-go/handler/blob/f96ffdde846be75dd40541aebd1dba604f274817/graphiql.go#L21
var graphiqlTemplate = htmltemplate.Must(htmltemplate.New("graphiql").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>{{ .Title }}</title>
	<style>
		body {
			padding: 0;
			margin: 0;
			min-height: 100vh;
		}
		#root {
			height: 100vh;
		}
	</style>
	<script
		crossorigin
		src="https://unpkg.com/react@17.0.2/umd/react.production.min.js"
	></script>
	<script
		crossorigin
		src="https://unpkg.com/react-dom@17.0.2/umd/react-dom.production.min.js"
	></script>
	<link rel="stylesheet" href="https://unpkg.com/graphiql@2.0.13/graphiql.min.css" />
</head>
<body>
	<div id="root">Loading...</div>
	<script
		crossorigin
		src="https://unpkg.com/graphiql@2.0.13/graphiql.min.js"
	></script>
	<script>
		// Collect the URL parameters
		var parameters = {};
		window.location.search.substr(1).split('&').forEach(function (entry) {
			var eq = entry.indexOf('=');
			if (eq >= 0) {
				parameters[decodeURIComponent(entry.slice(0, eq))] =
				decodeURIComponent(entry.slice(eq + 1));
			}
		});
		// Produce a Location query string from a parameter object.
		function locationQuery(params) {
			return '?' + Object.keys(params).filter(function (key) {
				return Boolean(params[key]);
			}).map(function (key) {
				return encodeURIComponent(key) + '=' +
				encodeURIComponent(params[key]);
			}).join('&');
		}
		// Derive a fetch URL from the current URL, sans the GraphQL parameters.
		var graphqlParamNames = {
			query: true,
			variables: true,
			operationName: true
		};
		var otherParams = {};
		for (var k in parameters) {
			if (parameters.hasOwnProperty(k) && graphqlParamNames[k] !== true) {
				otherParams[k] = parameters[k];
			}
		}
		var fetchURL = locationQuery(otherParams);
		// Defines a GraphQL fetcher using the fetch API.
		function graphQLFetcher(graphQLParams) {
			return fetch(fetchURL, {
				method: 'post',
				headers: {
					'Accept': 'application/json',
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(graphQLParams),
				credentials: 'include',
			}).then(function (response) {
				return response.text();
			}).then(function (responseBody) {
				try {
					return JSON.parse(responseBody);
				} catch (error) {
					return responseBody;
				}
			});
		}
		// When the query and variables string is edited, update the URL bar so
		// that it can be easily shared.
		function onEditQuery(newQuery) {
			parameters.query = newQuery;
			updateURL();
		}
		function onEditVariables(newVariables) {
			parameters.variables = newVariables;
			updateURL();
		}
		function onEditOperationName(newOperationName) {
			parameters.operationName = newOperationName;
			updateURL();
		}
		function updateURL() {
			history.replaceState(null, null, locationQuery(parameters));
		}
		ReactDOM.render(
			React.createElement(GraphiQL, {
				fetcher: graphQLFetcher,
				onEditQuery: onEditQuery,
				onEditVariables: onEditVariables,
				onEditOperationName: onEditOperationName,
				query: {{ .QueryString }},
				response: {{ .ResultString }},
				variables: {{ .VariablesString }},
				operationName: {{ .OperationName }},
			}),
			document.getElementById("root"),
		);
	</script>
</body>
</html>
`))

// graphiqlData is the page data structure of the rendered GraphiQL page
type graphiqlData struct {
	Title           string
	QueryString     string
	VariablesString string
	OperationName   string
	ResultString    string
}

type GraphiQL struct {
	Title   string
	Schema  *graphql.Schema
	Context context.Context
}

func (g *GraphiQL) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	opts := handler.NewRequestOptions(r)
	params := graphql.Params{
		Schema:         *g.Schema,
		RequestString:  opts.Query,
		VariableValues: opts.Variables,
		OperationName:  opts.OperationName,
		Context:        g.Context,
	}

	// Create variables string
	vars, err := json.MarshalIndent(params.VariableValues, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	varsString := string(vars)
	if varsString == "null" {
		varsString = ""
	}

	// Create result string
	var resString string
	if params.RequestString == "" {
		resString = ""
	} else {
		result, err := json.MarshalIndent(graphql.Do(params), "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resString = string(result)
	}

	d := graphiqlData{
		Title:           g.Title,
		QueryString:     params.RequestString,
		ResultString:    resString,
		VariablesString: varsString,
		OperationName:   params.OperationName,
	}

	err = graphiqlTemplate.Execute(w, d)
	if err != nil {
		panic(err)
	}
}
