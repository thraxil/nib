runserver:
	dev_appserver.py --port=5080 app.yaml

deploy:
	gcloud app deploy --project=appsembler-infrastructure
