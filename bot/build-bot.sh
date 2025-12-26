DOCKER_BUILDKIT=1 docker build -t nelchan:latest . \
--secret id=token,src=token.txt
