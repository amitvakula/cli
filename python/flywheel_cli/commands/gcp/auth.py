import sys
import urllib.parse

from ...errors import CliError
from ...sdk_impl import create_flywheel_session
from ...util import set_subparser_print_help
from .config import config


# TODO limit cloud-platform scope (healthcare scope not available yet)
SCOPES = ['userinfo.profile', 'userinfo.email', 'cloud-platform']
SCOPE = ' '.join('https://www.googleapis.com/auth/' + scope for scope in SCOPES)


def add_command(subparsers):
    parser_help = 'Manage GCP access-tokens stored in Flywheel'
    parser = subparsers.add_parser('auth',
        help=parser_help, description=parser_help + '.')
    auth_subparsers = parser.add_subparsers(
        title='available gcp auth commands', metavar='')

    parser_login_help = 'Login to Google and store access-token'
    parser_login = auth_subparsers.add_parser('login',
        help=parser_login_help, description=parser_login_help + '.')
    parser_login.set_defaults(func=auth_login)
    parser_login.set_defaults(parser=parser_login)

    parser_list_help = 'List GCP access-tokens'
    parser_list = auth_subparsers.add_parser('list',
        help=parser_list_help, description=parser_list_help + '.')
    parser_list.set_defaults(func=auth_list)
    parser_list.set_defaults(parser=parser_list)

    parser_logout_help = 'Remove GCP access-token (may affect GCP jobs)'
    parser_logout = auth_subparsers.add_parser('logout',
        help=parser_logout_help, description=parser_logout_help + '.')
    parser_logout.add_argument('id',
        help='Access-token ID')
    parser_logout.set_defaults(func=auth_logout)
    parser_logout.set_defaults(parser=parser_logout)

    set_subparser_print_help(parser, auth_subparsers)
    return parser


def auth_login(args):
    api = create_flywheel_session()
    api_url = api.baseurl.replace('/api', '')
    google_auth = api.get('/config').get('auth', {}).get('google')
    if not google_auth:
        raise CliError('Google auth not configured on ' + api_url)
    url = google_auth['auth_endpoint'] + '?' + urllib.parse.urlencode({
        'prompt': 'select_account',
        'access_type': 'offline',
        'scope': SCOPE,
        'redirect_uri': api_url + '/ghc',
        'response_type': 'code',
        'client_id': google_auth['client_id']
    })
    print('Please visit the following URL in your browser:')
    print('  ' + url)
    code = input('Enter the verification code: ')
    token = api.post('/users/self/tokens', json={'auth_type': 'google', 'code': code})
    print('Added GCP access-token for ' + token['identity']['email'])


def auth_list(args):
    api = create_flywheel_session()
    tokens = api.get('/users/self/tokens?' + urllib.parse.urlencode({'scope': SCOPE}))
    if tokens:
        print('Available GCP access-tokens:')
        for token in tokens:
            print('  {_id}: {identity[email]}'.format(**token))
    else:
        print('No GCP access-tokens added yet.')


def auth_logout(args):
    api = create_flywheel_session()
    api.delete('/users/self/tokens/' + args.id)
    print('Removed GCP access-token ' + args.id)


def get_token(token_id=None):
    api = create_flywheel_session()
    if token_id is None:
        token_id = get_token_id()
    return api.get('/users/self/tokens/' + token_id)['access_token']


def get_token_id():
    api = create_flywheel_session()
    tokens = api.get('/users/self/tokens?' + urllib.parse.urlencode({'scope': SCOPE}))
    if not tokens:
        # use first (most recently used) token by default
        raise CliError('Not logged in to GCP. Use `fw gcp auth login` first.')
    return tokens[0]['_id']
