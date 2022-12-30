# Téléchargement des recettes HelloFresh

## Contexte

HelloFresh offre un service de livraison de courses à domicile accompagnées de recettes.
Ce programme télécharge les recettes HelloFresh depuis le site HelloFresh belge.

## Compilation

```sh
go build -o recettes-hellofresh
```

## Pré-requis

* une instance Nextcloud (ou n'importe quel serveur WebDAV)

## Usage

```sh
podman-compose up -d
./recettes-hellofresh config.yaml
```
