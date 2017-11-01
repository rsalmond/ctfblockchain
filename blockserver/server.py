from flask import Flask, jsonify, request
import flask_sqlalchemy


db = flask_sqlalchemy.SQLAlchemy()
app = Flask(__name__)
app.config['SQLALCHEMY_DATABASE_URI'] = 'sqlite:///blocks.db'

db.init_app(app)

class Block(db.Model):
    id = db.Column(db.Integer, primary_key=True, autoincrement=True)
    identifier = db.Column(db.String)
    nonce = db.Column(db.Integer)
    data = db.Column(db.String)
    previous_hash = db.Column(db.String)
    difficulty = db.Column(db.Integer)

    def save(self):
        db.session.add(self)
        db.session.commit()

    @classmethod
    def all(cls):
        return db.session.query(cls).all()

    @classmethod
    def chain(cls):
        blocks = db.session.query(cls).order_by(cls.id).all()
        return [block.serializable() for block in blocks]

    def serializable(self):
        blob = {}
        for attr in ('identifier', 'nonce', 'data', 'previous_hash'):
            blob[attr] = getattr(self, attr)

        return blob


@app.route('/chain', methods=['GET', 'POST'])
def chain():
    if request.method == 'GET':
        return jsonify(Block.chain())
        #return jsonify(Block.chain()[0])
