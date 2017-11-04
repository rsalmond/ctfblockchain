from flask import Flask, jsonify, request
from datetime import datetime as dt
import flask_sqlalchemy
import hashlib
import logging
import sys

db = flask_sqlalchemy.SQLAlchemy()
app = Flask(__name__)
app.config['SQLALCHEMY_DATABASE_URI'] = 'sqlite:///blocks.db'
app.logger.addHandler(logging.StreamHandler(sys.stdout))
app.logger.setLevel(logging.INFO)

db.init_app(app)

class Status(db.Model):
    id = db.Column(db.Integer, primary_key=True, autoincrement=True)
    client_id = db.Column(db.String, unique=True)
    username = db.Column(db.String)
    updated_at = db.Column(db.DateTime, default=dt.utcnow)
    hashrate = db.Column(db.Integer)

    def save(self):
        db.session.add(self)
        db.session.commit()

    @classmethod
    def from_client_id(cls, client_id):
        return db.session.query(cls).filter(cls.client_id==client_id).first()

    @classmethod
    def validate(cls, status):
        for attr in ('username', 'client_id', 'hashrate'):
            if not status.get(attr):
                return False

        if not isinstance(status.get('username'), basestring):
            return False

        if not isinstance(status.get('client_id'), basestring):
            return False

        if not isinstance(status.get('hashrate'), int):
            return False

        return True

    @classmethod
    def update(cls, status):
        if not cls.validate(status):
            app.logger.info(status)
            app.logger.info('Invalid status message received, discarding.')
            return False

        registered_status = Status.from_client_id(status.get('client_id'))
        if registered_status:
            registered_status.updated_at = dt.utcnow()
            registered_status.hashrate = status.get('hashrate')
            registered_status.save()
            app.logger.info('Updated status for {}/{} with hashrate {}'.format(registered_status.username, \
                    registered_status.client_id, \
                    registered_status.hashrate))
        else:
            new_status = Status()
            new_status.client_id = status.get('client_id')
            new_status.username = status.get('username')
            new_status.hashrate = status.get('hashrate')
            new_status.save()
            app.logger.info('Registered new status for {}/{} with hashrate {}'.format(new_status.username, \
                    new_status.client_id, \
                    new_status.hashrate))

        return True

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
        return message.hexdigest()

    @classmethod
    def update(cls, block):
        #TODO: more here
        if Block.validate(block):
            if Block.verify_hash(block):
                oldblock = db.session.query(Block).filter(Block.data==block.get('data')).first()
                if oldblock.nonce:
                    # nonce has already been found, skipping
                    app.logger.info('Nonce already found for block {}, discarding.'.format(block))
                    return
                else:
                    oldblock.nonce = block.get('nonce')
                    oldblock.identifier = block.get('identifier')
                    oldblock.previous_hash = block.get('previous_hash')
                    oldblock.save()
                    return True
            else:
                app.logger.info('Invalid hash for block {}, discarding.'.format(block))
                return
        else:
            app.logger.info('Invalid block {}, discarding.'.format(block))
            return

@app.route('/chain', methods=['GET', 'POST'])
def chain():
    if request.method == 'GET':
        return jsonify(Block.chain())
    elif request.method == 'POST':
        if request.json is not None:
            if isinstance(request.json, list):
                updated = False
                for block in request.json:
                    if Block.update(block):
                        updated = True
                if updated:
                    return 'blockchain updated'
                else:
                    return 'blockchain not updated'
            else:
                return "invalid json", 400
        else:
            return "invalid post", 400


@app.route('/status', methods=['GET', 'POST'])
def status():
    if request.method == 'GET':
        pass
    elif request.method == 'POST':
        if request.json is not None:
            if isinstance(request.json, dict):
                if Status.update(request.json):
                    return 'status updated'
                else:
                    return 'error updating status', 500
            else:
                return 'invalid json', 400
        else:
            return 'invalid post', 400
