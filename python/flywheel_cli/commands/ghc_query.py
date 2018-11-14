import sys

from flywheel.rest import ApiException
from ..sdk_impl import create_flywheel_client, SdkUploadWrapper
from ..config import GCPConfig


def add_command(subparsers):
    parser = subparsers.add_parser('query', help='Query dicom files using BigQuery')
    parser.add_argument('where', metavar='COND', nargs='*', help='Filter conditions (SQL WHERE clause')
    parser.add_argument('--study', action='store_true', help='Only show studies')

    parser.set_defaults(func=ghc_query)
    parser.set_defaults(parser=parser)

    return parser


def ghc_query(args):
    ghc_config = GCPConfig()

    core_keys = ('project', 'token')
    bq_keys = ('dataset', 'table')

    payload = {key: ghc_config['core'][key] for key in core_keys if ghc_config.get('core', {}).get(key)}
    for key in bq_keys:
        if ghc_config.get('bigquery', {}).get(key):
            payload[key] = ghc_config['bigquery'][key]


    if args.where:
        payload['where'] = ' '.join(args.where)

    print('Running query...')
    fw = SdkUploadWrapper(create_flywheel_client())
    try:
        resp = fw.call_api('/gcp/hc/query', 'POST', body=payload, response_type=object)
    except ApiException as e:
        print('{} {}: {}'.format(e.status, e.reason, e.detail))
        sys.exit(1)
    print_results(resp, studies_only=args.study)
    ghc_config.update_section('bigquery', {'last_query_id': resp['query_id']})
    ghc_config.save()


def print_results(result, studies_only=False):
    print('Query: {query_id}\n'
          'Number of matching studies: {total_studies} / series: {total_series} / instances: {total_instances}'
          .format(**result))

    if studies_only:
        study_template = ' {StudyInstanceUID} ({StudyDate}, {StudyDescription})\n'
    else:
        study_template = ' {StudyInstanceUID} ({StudyDate}, {StudyDescription}, {series_count} series)\n'
        series_template = ' {SeriesInstanceUID} ({SeriesDescription}, {instance_count} instances)\n'

    # tree formatting prefixes
    PFX_SINGLE   = '──'
    PFX_FIRST    = '┬──'
    PFX_MID      = '├──'
    PFX_LAST     = '└──'
    PAR_PFX_MID  = '│   '
    PAR_PFX_LAST = '    '

    for study_num, study in enumerate(result['studies']):
        last_study = study_num == result['study_count'] - 1
        if result['study_count'] == 1:
            prefix = PFX_SINGLE
        elif study_num == 0:
            prefix = PFX_FIRST
        elif last_study:
            prefix = PFX_LAST
        else:
            prefix = PFX_MID
        sys.stdout.write(prefix + study_template.format(**study))

        if not studies_only:
            for series_num, series in enumerate(study['series']):
                last_series = series_num == study['series_count'] - 1

                prefix = PAR_PFX_LAST if last_study else PAR_PFX_MID
                prefix += PFX_LAST if last_series else PFX_MID
                sys.stdout.write(prefix + series_template.format(**series))
