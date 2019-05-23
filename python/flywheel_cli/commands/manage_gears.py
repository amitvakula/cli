import docker
import json
import logging
import os
import sys
import yaml

import flywheel
from .. import sdk_impl

CONFIG_PATH = '~/.config/flywheel/gears'

log = logging.getLogger(__name__)


def add_commands(subparsers, parsers):
    # Get Gears
    gears_pull_parser = subparsers.add_parser('pull', help='Pull site gears to inventory')
    gears_pull_parser.add_argument('--filter-site', action='append', help='filter sites to pull from')
    gears_pull_parser.add_argument('--inventory-file', '-f', default=os.path.expanduser(os.path.join(CONFIG_PATH, 'inventory.yml')), help='Path to the inventory file')
    gears_pull_parser.set_defaults(func=gears_pull)

    # Print out gears
    gear_print_parser = subparsers.add_parser('print', help='Print out inventory')
    gear_print_parser.add_argument('--filter-site', action='append', help='filter sites to push to')
    gear_print_parser.add_argument('--filter-gear', action='append', help='filter gears to push')
    gear_print_parser.add_argument('--all-versions', action='store_true', help='Show all versions installed on site')
    gear_print_parser.add_argument('--inventory-file', '-f', default=os.path.expanduser(os.path.join(CONFIG_PATH, 'inventory.yml')), help='Path to the inventory file')
    gear_print_parser.set_defaults(func=gears_print)

    # Push Gears
    gear_push_parser = subparsers.add_parser('push', help='Push site gears from inventory to sites')
    gear_push_parser.add_argument('--filter-site', action='append', help='filter sites to push to')
    gear_push_parser.add_argument('--filter-gear', action='append', help='filter gears to push')
    gear_push_parser.add_argument('--inventory-file', '-f', default=os.path.expanduser(os.path.join(CONFIG_PATH, 'inventory.yml')), help='Path to the inventory file')
    gear_push_parser.add_argument('--gear-manifest', '-g', help='Path to a gear manifest')
    gear_push_parser.set_defaults(func=gears_push)


def gears_print(args):
    """Print out inventory"""
    inventory = _load_inventory(args.inventory_file)
    for site in inventory['sites']:
        if not args.filter_site or site['name'] in args.filter_site:
            print('Site:', site['name'])
            for gear_name, gear in inventory[site['name']].items():
                if not args.filter_gear or gear_name in args.filter_gear:
                    print('\tGear:', gear_name)
                    if args.all_versions:
                        print('\t\tVersions:', gear['versions'])
                    else:
                        print('\t\tLatest Version:', gear['latest-version'])

def gears_push(args):
    """Push push-version to site"""
    inventory = _load_inventory(args.inventory_file)
    for site in inventory['sites']:
        if not args.filter_site or site['name'] in args.filter_site:
            client = flywheel.Client(site['api-key'])
            _push_local_gears(client, inventory.get(site['name']), args.filter_gear)


def _push_local_gears(client, site_gears, gear_filters):
    for gear_name, gear in site_gears.items():
        if gear.get('push-version') and gear['push-version'] not in gear.get('versions', []):
            push_gear(client, gear['manifest'])


def push_gear(client, gear_manifest_uri):
    user = client.get_current_user()

    # Create docker client
    docker_client = docker.from_env()

    # Load in the gear manifest
    manifest = load_manifest(gear_manifest_uri)

    # Pull the manifest image
    image_repo = manifest['custom']['gear-builder']['image']
    if ':' in image_repo:
        image_repo, image_tag = image_repo.split(':')
    else:
        image_tag = None
    docker_client.images.pull(image_repo, tag=image_tag)

    # Retrieve the gear ticket
    gear_ticket_id = client.prepare_add_gear(manifest)

    # Login to the flywheel registry
    domain = '/'.join(fw.get_config().site.api_url.split('/')[2:-1])
    apikey = client.api_client.configuration.api_key['Authorization']
    password = domain + ':' + apikey
    docker_client.login(username=user.email,
                        password=password,
                        registry=domain)
    try:
        log.debug('Pushing gear...')
        digest = json.loads(docker_client.images.push(image_repo, tag=image_tag, decode=True).split('\r\n')[-2])['aux']['Digest']
    except Exception as e:
        log.error('Failed to push image', exc_info=True)
        return

    ticket = {
        'ticket': gear_ticket_id,
        'repo': domain + '/' + image_repo + ':' + image_tag,
        'pointer': digest
    }
    client.save_gear(ticket)


def load_manifest(gear_manifest_uri):
    doc = {}
    if gear_manifest_uri.startswith('https'):
        pass
    else:
        with open(gear_manifest_uri, 'r') as gear_manifest:
            doc = json.load(gear_manifest)
    return doc


def gears_pull(args):
    """Pull site gears info to local inventory"""
    inventory = _load_inventory(args.inventory_file)
    for site in inventory['sites']:
        client = flywheel.Client(site['api-key'])
        _pull_site_gears(client, inventory.setdefault(site['name'], {}))
    _save_inventory(args.inventory_file, inventory)


def _pull_site_gears(client, gears):
    for gear in client.get_all_gears():
        gear_inventory = {}

        # Get all installed versions of the gear
        gear_inventory['versions'] = [gear_version.gear.version for gear_version in client.get_all_gears(all_versions=True, filter='gear.name={}'.format(gear.gear.name))]

        # If the latest version on the site is different than the latest version in the manifest set the out-of-date flag
        gear_inventory['latest-version'] = gear.gear.version
        gears.setdefault(gear.gear.name, {}).update(gear_inventory)


def _load_inventory(inventory_path):
    """Loads in the inventory file"""
    with open(inventory_path, 'r') as inventory_file:
        inventory = yaml.safe_load(inventory_file)
    return inventory


def _save_inventory(inventory_path, inventory):
    with open(inventory_path, 'w') as inventory_file:
        yaml.safe_dump(inventory, inventory_file, default_flow_style=False)

