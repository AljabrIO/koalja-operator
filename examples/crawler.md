# crawler.yaml

This pipeline shows how to build a web crawler in Koalja.

Note that this example uses an [ArangoDB deployment](https://github.com/arangodb/kube-arangodb)
as its underlying database.

The pipeline starts by fetching URL's from a database.
The query used to fetch these URL's ensures that it will only fetch
new URL's (compared to what it has fetched before).

These URL's are send to a crawl task. The crawler fetches the content of the URL
and extracts all URL's it find in that content.
The list of distinct URL's found in the content are send to the next task
as a single file or with one URL per line.

The generated files with one URL per line are split into individual URL's.

Finally the individual URL's are stored back into the database so they can
be crawled as well.

## Status

This pipeline is missing the final step to store the crawled URL's back into
the database. For the rest it is fully functional.
