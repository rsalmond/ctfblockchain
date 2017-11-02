from flask import Flask, jsonify, request
import flask_sqlalchemy
import hashlib


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

    attrs = ('identifier', 'nonce', 'data', 'previous_hash', 'difficulty')

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
        for attr in self.attrs:
            blob[attr] = getattr(self, attr)

        return blob
    
    @classmethod
    def validate(cls, block):
        """ check an incoming block for validity """
        for attr in cls.attrs:
            if block.get(attr) is None:
                return False

        if not isinstance(block.get('nonce'), int):
            return False

        if not isinstance(block.get('identifier'), basestring):
            return False

        if not isinstance(block.get('data'), basestring):
            return False

        if not isinstance(block.get('previous_hash'), basestring):
            return False

        # only the genesis block should have None for prev hash
        if block.get('identifier') != u'000102030405060708090A0B0C0D0E0F':
            if block.get('previous_hash') == u'None':
                return False

        return True
    
    @classmethod
    def verify_hash(cls, block):
        message = hashlib.sha256()
        message.update(str(block.get('identifier')).encode('utf-8'))
        message.update(str(block.get('nonce')).encode('utf-8'))
        message.update(str(block.get('data')).encode('utf-8'))
        message.update(str(block.get('previous_hash')).encode('utf-8'))
        import pdb
        pdb.set_trace()
        return message.hexdigest()

    @classmethod
    def update(cls, block):
        #TODO: more here
        if Block.validate(block):
            if Block.verify_hash(block):
                oldblock = db.session.query(Block).filter(Block.data==request.json[0].get('data')).first()
            else:
                return "invalid hash"
        else:
            return "invalid block"

@app.route('/chain', methods=['GET', 'POST'])
def chain():
    if request.method == 'GET':
        return jsonify(Block.chain())
    elif request.method == 'POST':
        if request.json is not None:
            if isinstance(request.json, list):
                for block in request.json:

            else:
                return "invalid json"
        else:
            return "invalid post"
                        
