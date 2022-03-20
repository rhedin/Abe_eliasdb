EliasDB Import/Export
=====================

EliasDB supports importing and exporting of data in various ways:
- By [embedding](embedding.md) EliasDB in another Go project.
- By using the [REST API](http://petstore.swagger.io/?url=https://devt.de/krotik/eliasdb/raw/master/swagger.json) interface.
- By running an [ECAL](http://petstore.swagger.io/?url=https://devt.de/krotik/eliasdb/raw/master/swagger.json) script.
- By running the `EliasDB` executable with import/export parameters in the CLI.

Bulk importing and exporting is best done through the last option.

Bulk importing and exporting via the CLI
--
Bulk import/export through the CLI is available using the `eliasdb` binary with the `server` command. In general there are two different types of import/export modes:
- Normal import/export through a single compact ZIP file.
- Large scale import/export though multiple ZIP files.

Parameter|Description
-|-
-export|Export the current DB into a ZIP file. The data of each partition is stored into a separate file as a JSON object.
-import|Import into the current DB from a ZIP file. The data is expected in the same format as in the `-export` case.
-export-ls|Export the current DB into multiple ZIP file. The data of each partition is stored into two separate files for nodes and edges in a line-delimited JSON format.
-import-ls|Import into the current DB from a ZIP file. The data is expected in the same format as in the `-export-ls` case.

By default the server will start after the import/export operation. This can be disabled by using the `-no-serv` parameter.

Format for normal import/export
--
The normal import/export will work on a single ZIP file which contains a series of `.json` files.

```
eliasdb server -export mydb.zip -no-serv
eliasdb server -import mydb.zip -no-serv
```

The name of each file will become a separate partition. Each of these `.json` files contains a single JSON object with the following structure:
```
{
  nodes : [ { <attr> : <value>, ... }, ... ]
  edges : [ { <attr> : <value>, ... }, ... ]
}
```
When embedding EliasDB in another Go project this can be produced and consumed via `graph.ExportPartition` and `graph.ImportPartition`.

Format for large scale import/export
--
The large scale
