# Downloader

Script used to download profiles from the GCS bucket

## Prerequisites

* list and read permission for the `sentryio-profiles` bucket

## List of profiles to download

In order to download the profiles it's necessary to first create a list of `gs` paths of the profiles we want to download.

### How To

1. authenticate (for gsutil CLI): `gcloud auth login`
2. set the sentryio project: `gcloud config set project sentryio`
3. save gs profiles path to a file: `gsutil ls gs://sentryio-profiles/{org_id}/{project_id}/ | head -n {num_of_profiles_we_want} > profiles_list.txt`

## Download the profiles

### How To

1. obtain credentials and put them in a well known location for *Application Default Credential*: `gcloud auth application-default login`
2. create a folder where the profiles will be stored: `mkdir profiles`
3. build the downloader: `make downloader`
4. run the downloader: `./downloader ./profiles_list.txt ./profiles`
