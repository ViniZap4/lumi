# Tailscale com a configuração do Serve dentro da imagem.
# O job do deploy fala com o Docker do servidor de dentro de um contêiner:
# um bind mount de arquivo do job apontaria para um caminho que o host não tem
# (mesma lição do porvia, ADR 0048).
ARG TAILSCALE_IMAGE=tailscale/tailscale:v1.104.7
FROM ${TAILSCALE_IMAGE}
ARG SERVE
COPY ${SERVE}.json /config/serve.json
