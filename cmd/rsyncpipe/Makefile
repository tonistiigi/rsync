builder:
	docker build -t tonistiigi/rsyncpipe:builder -f Dockerfile.build .

dist: builder
	mkdir -p dist
	docker run --rm tonistiigi/rsyncpipe:builder tar cvf - rsync rsyncpipe Dockerfile | tar -xvf - -C ./dist 

distimage: dist
	docker build -t tonistiigi/rsyncpipe:latest dist