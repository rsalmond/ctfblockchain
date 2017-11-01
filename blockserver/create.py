def create():
    global app
    app.extensions.get('sqlalchemy').db.create_all()
