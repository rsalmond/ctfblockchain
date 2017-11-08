import sys
import time
import json
import random
import string
import hashlib

chain_data = {
    0: {'difficulty': 8, 'data': None},
    1: {'difficulty': 4, 'data': 'Four be the things I am wiser to know:'},
    2: {'difficulty': 4, 'data': 'Idleness, sorrow, a friend, and a foe.'},
    3: {'difficulty': 5, 'data': 'Four be the things I\'d been better without:'},
    4: {'difficulty': 6, 'data': 'Love, curiosity, freckles, and doubt.'},
    5: {'difficulty': 7, 'data': 'Three be the things I shall never attain:'},
    6: {'difficulty': 9, 'data': 'Envy, content, and sufficient champagne.'},
    7: {'difficulty': 11, 'data': 'Three be the things I shall have till I die:'},
    8: {'difficulty': 13, 'data': 'Laughter and hope and a sock in the eye.'},
    9: {'difficulty': 16, 'data': ' -Dorothy Parker'}
    }

def loadchain():
    with open('chainfile.json', 'r') as f:
        chain = json.load(f)

    return chain

def gen_blockid():
    return ''.join([random.choice(string.hexdigits.upper()) for x in range(32)])

def hash_block(block):
    message = hashlib.sha256()
    message.update(str(block['identifier']).encode('utf-8'))
    message.update(str(block['nonce']).encode('utf-8'))
    message.update(str(block['data']).encode('utf-8'))
    message.update(str(block['previous_hash']).encode('utf-8'))
    return message.hexdigest()

def gen_block(chain, block_id):
    previous = chain[-1:].pop()
    block_num = len(chain)
    block = {}
    block['identifier'] = block_id
    block['data'] = chain_data[block_num].get('data')
    block['previous_hash'] = hash_block(previous)
    return block


if __name__ == '__main__':
    chain = loadchain()
    #for block in gen_block(chain, gen_blockid()):
    while True:
        block = gen_block(chain, gen_blockid())
        difficulty = chain_data[len(chain)].get('difficulty')
        difficulty_target = '0' * difficulty
        start = int(time.time())
        count = 0
        while True:
            count += 1
            block['nonce'] = random.randint(1,100000000)
            hashed = hash_block(block)
            if hashed[0:difficulty] == difficulty_target:
                print(block)
                print(hashed)
                chain.append(block)
                break

            if int(time.time()) >  start + 10:
                print('Hashes per second: {}'.format(count / 10))
                start = int(time.time())
                count = 0
