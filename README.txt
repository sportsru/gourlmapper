About:
------

gourlmapper – simple and fast service for url mapping with nginx & redis
(support internal/external redirects)

* usable not only with nginx probably


Description
------------

Use url maps for redirect from one url to another. 

Supported redirect types:
* external 301 (Moved permanently)
* internal (nginx X-Accel-Redirect header)

Use storages:
* local – file slurped in memory (be careful with it's size) for fast permanent mapping (optional)
* redis (for dynamic urls), redis values caches locally for efficiency (required, TODO: optional)

Support mutiply hosts, but map "host -> prefix" hardcoded in sources (TODO: move to config)

Usage:
-------

   ./gourlmapper -h

Example:

    GOMAXPROCS=2 ./gourlmapper -f gen_map.txt

Environment
------

 - install Redis (redis.io)
 – install and setup nginx
 - configure nginx
 – add keys in redis
 – *optional* create/generate url's map file

Local file format:
-------------------

<prefix> <redir_type> <from> <to>

    ru I /1000000 /_tags/ru/1000000.html
    ru 301 /tags/ru/1000000.html /1000000

Redis keys format:
------------------

    $ redis-cli

# set internal redirects:
    > SET "ru:/dynamo" "I /_tags/1685245.html"
    > SET "ua:/dynamo" "I /_tags/1047809.html"

# set external redirects:
    > SET  "ru:/tags/1685245.html" "301 /dynamo"
    > SET  "ua:/tags/1047809.html" "301 /dynamo"


Nginx config example:
--------------------

nginx.conf:

 	log_format timed_combined '$remote_addr - $remote_user [$time_local]  '
    	'"$request" $status $body_bytes_sent '
    	'$request_time $upstream_response_time $pipe';
	access_log log/hru.access.log timed_combined;

	# one host with 1st map
	server {
	    listen       8069;
	    server_name  ru.localhost;
	    proxy_set_header        Host            www.sports.ru;
	    include conf/common_hru.conf;
	}

	# 2nd host with 2nd map
	server {
	    listen       8069;
	    server_name  ua.localhost;
	    proxy_set_header        Host            ua.tribuna.com;
	    include conf/common_hru.conf;
	}

common_hru.conf:

	# all other rules must be on top
    location = /favicon.ico {
        root    htdocs/favicon.ico;
    }

    # only for internal redirects
    location ^~ /_tags/(\d+)\.html$ {
        internal;
        set $tag_id $1;
        rewrite ^/(.*) /tags/$tag_id.html;
        error_page 418 = @tag;
        return 418;
    }
    # -> target 
    location @tag {
        proxy_pass http://www.sports.ru;
    }

    # our fancy redirector here:
    location / {
        proxy_pass http://127.0.0.1:8080;
    }


How to test
-------------


bare service:

./gourlmapper
curl -I http://127.0.0.1:8080/dynamo -H "Host: www.sports.ru"


with nginx:

./gourlmapper
curl -I http://ua.localhost:8069/dynamo 
curl -I http://ua.localhost:8069/


Other notes:
-------------

available hosts hardcoded in `gourlmapper/main.go` by map `hostsMap`

TODO:
----------------

- optional redis
- add statistic
- test, check go test -race 




