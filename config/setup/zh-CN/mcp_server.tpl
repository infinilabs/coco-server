POST $[[SETUP_INDEX_PREFIX]]mcp-server$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/d04vm873edbo4f7e6stg
{
	"id": "d04vm873edbo4f7e6stg",
	"created": "2025-04-24T16:49:36.1654+08:00",
	"updated": "2025-04-24T17:30:27.387422+08:00",
	"name": "filesystem",
	"icon": "font_filetype-folder",
	"type": "stdio",
	"category": "FileSystem",
	"config": {
		"args": ["-y", "@modelcontextprotocol/server-filesystem", "~/Desktop"],
		"command": "npx",
         "env": {}
	},
	"enabled": true
}

POST $[[SETUP_INDEX_PREFIX]]mcp-server$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/d053quf3edbhe0kp9gd0
{
	"id": "d053quf3edbhe0kp9gd0",
	"created": "2025-04-24T21:32:41.163768+08:00",
	"updated": "2025-04-24T21:32:41.163778+08:00",
	"name": "sequential-thinking",
	"icon": "font_robot",
	"type": "stdio",
	"category": "Basic",
	"config": {
		"args": ["-y", "@modelcontextprotocol/server-sequential-thinking"],
		"command": "npx",
		"env": {}
	},
	"enabled": true
}

POST $[[SETUP_INDEX_PREFIX]]mcp-server$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/d054rin3edbhiauqki7g
{
	"id": "d054rin3edbhiauqki7g",
	"created": "2025-04-24T22:42:18.66328+08:00",
	"updated": "2025-04-24T22:57:08.206049+08:00",
	"name": "playwright",
	"icon": "font_robot",
	"type": "stdio",
	"category": "Network Tools",
	"config": {
		"args": ["@playwright/mcp@latest", "--headless"],
		"command": "npx",
		"env": {}
	},
	"enabled": true
}
