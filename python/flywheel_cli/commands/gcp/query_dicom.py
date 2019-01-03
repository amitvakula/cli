import argparse
import itertools
import sys


from ...errors import CliError
from ...sdk_impl import create_flywheel_session
from .auth import get_token
from .flywheel_gcp import GCP, GCPError
from .profile import get_profile


QUERY_DICOM_DESC = """
Run assisted SQL query (only WHERE clase needed) on a BigQuery table exported
from Healthcare API. Assuming that Healthcare API dataset/dicomstore matches
BigQuery dataset/table name.

If a BigQuery table with the same name as the dicomstore doesn't exist yet
(or using `--export`), then the dicomstore is exported before the query.

examples:
  fw gcp query dicom --dataset ds --dicomstore ds 'AccessionNumber="test"'
  fw gcp query dicom 'PatientSex="M" and PatientAge>"040Y"'
"""

# get main cols of the 1st 100 series matching a given where clause
SQL_TEMPLATE = """
SELECT StudyInstanceUID, SeriesInstanceUID,
  COUNT(DISTINCT SOPInstanceUID) AS image_count,
  MIN(AccessionNumber) AS AccessionNumber,
  MIN(PatientID) AS PatientID,
  MIN(StudyID) AS StudyID,
  MIN(StudyDate) AS StudyDate,
  MIN(StudyTime) AS StudyTime,
  MIN(StudyDescription) AS StudyDescription,
  MIN(SeriesDate) AS SeriesDate,
  MIN(SeriesTime) AS SeriesTime,
  MIN(SeriesDescription) AS SeriesDescription
FROM {dataset}.{table}
WHERE {where}
GROUP BY StudyInstanceUID, SeriesInstanceUID
ORDER BY StudyInstanceUID, SeriesInstanceUID
"""


def add_command(subparsers):
    parser = subparsers.add_parser('dicom',
        help='Query dicoms in Healthcare API via BigQuery',
        description=QUERY_DICOM_DESC,
        formatter_class=argparse.RawTextHelpFormatter)

    profile = get_profile()
    project = profile.get('project')
    location = profile.get('location')
    dataset = profile.get('hc_dataset')
    dicomstore = profile.get('hc_dicomstore')

    parser.add_argument('--project', metavar='NAME', default=project,
        help='GCP project (default: {})'.format(project))
    parser.add_argument('--location', metavar='NAME', default=location,
        help='Location (default: {})'.format(location))
    parser.add_argument('--dataset', metavar='NAME', default=dataset,
        help='Dataset (default: {})'.format(dataset))
    parser.add_argument('--dicomstore', metavar='NAME', default=dicomstore,
        help='Dicomstore / table (default: {})'.format(dicomstore))
    parser.add_argument('--export', action='store_true',
        help='Export to BigQuery first (even if table exists)')
    parser.add_argument('sql', metavar='SQL WHERE', nargs=argparse.REMAINDER,
        help='SQL WHERE clause')
    parser.add_argument('--studies', action='store_true',
        help='Return studies only (hide series)')
    parser.add_argument('--uids', action='store_true',
        help='Return UIDs only (useful for import)')

    parser.set_defaults(func=query_dicom)
    parser.set_defaults(parser=parser)
    return parser


def query_dicom(args):
    for param in ['project', 'location', 'dataset', 'dicomstore']:
        if not getattr(args, param, None):
            raise CliError(param + ' required')

    gcp = GCP(get_token)
    if args.export or args.dicomstore not in gcp.bq.list_tables(args.project, args.dataset):
        print('Exporting Healthcare API dicomstore to BigQuery')
        gcp.hc.export_to_bigquery(args.project, args.location, args.dataset, args.dicomstore)

    # TODO enable raw queries (?)
    query = SQL_TEMPLATE.format(dataset=args.dataset, table=args.dicomstore, where=' '.join(args.sql) or '1=1')
    try:
        result = gcp.bq.run_query(args.project, query)
    except GCPError as ex:
        raise CliError(str(ex))
    hierarchy = result_to_hierarchy(result)
    summary = 'Query matched {total_studies} studies, {total_series} series, {total_images} images'.format(**hierarchy)
    print(summary, file=sys.stderr)

    if args.uids:
        print_uids(hierarchy, studies_only=args.studies)
    else:
        print_tree(hierarchy, studies_only=args.studies)


def result_to_hierarchy(result):
    total_studies = 0
    total_series = 0
    total_images = 0
    studies = []

    for _, rows in itertools.groupby(result['rows'], key=lambda row: row['StudyInstanceUID']):
        total_studies += 1
        series = []
        for row in rows:
            total_series += 1
            total_images += int(row['image_count'])
            series.append({
                'SeriesDate': row['SeriesDate'],
                'SeriesTime': row['SeriesTime'],
                'SeriesInstanceUID': row['SeriesInstanceUID'],
                'SeriesDescription': row['SeriesDescription'],
                'image_count': int(row['image_count']),
            })

        # use last iterated `row` to get study properties
        studies.append({
            'StudyDate': row.get('StudyDate'),
            'StudyTime': row.get('StudyTime'),
            'StudyInstanceUID': row.get('StudyInstanceUID'),
            'StudyDescription': row.get('StudyDescription'),
            'series_count': len(series),
            'series': sorted(series, key=lambda s: (s['SeriesDate'], s['SeriesTime']), reverse=True),
            'subject': row['PatientID'].rpartition('@')[0] or 'ex' + row['StudyID'],
        })

    return {
        'query_id': result['query_id'],
        'total_studies': total_studies,
        'total_series': total_series,
        'total_images': total_images,
        'studies': sorted(studies, key=lambda s: (s['StudyDate'], s['StudyTime']), reverse=True),
    }


def print_uids(hierarchy, studies_only=False):
    for study in hierarchy['studies']:
        if studies_only:
            print(study['StudyInstanceUID'])
        else:
            for series in study['series']:
                print(series['SeriesInstanceUID'])


def print_tree(hierarchy, studies_only=False):
    if studies_only:
        study_template = ' {StudyInstanceUID} ({StudyDate}, {StudyDescription})\n'
    else:
        study_template = ' {StudyInstanceUID} ({StudyDate}, {StudyDescription}, {series_count} series)\n'
        series_template = ' {SeriesInstanceUID} ({SeriesDescription}, {image_count} images)\n'

    # tree formatting prefixes
    PFX_SINGLE   = '──'
    PFX_FIRST    = '┬──'
    PFX_MID      = '├──'
    PFX_LAST     = '└──'
    PAR_PFX_MID  = '│   '
    PAR_PFX_LAST = '    '

    last_study_num = hierarchy['total_studies'] - 1
    for study_num, study in enumerate(hierarchy['studies']):
        last_study = study_num == last_study_num
        if last_study_num == 0:
            prefix = PFX_SINGLE
        elif study_num == 0:
            prefix = PFX_FIRST
        elif last_study:
            prefix = PFX_LAST
        else:
            prefix = PFX_MID

        sys.stdout.write(prefix + study_template.format(**study))

        if not studies_only:
            last_series_num = study['series_count'] - 1
            for series_num, series in enumerate(study['series']):
                last_series = series_num == last_series_num
                prefix = PAR_PFX_LAST if last_study else PAR_PFX_MID
                prefix += PFX_LAST if last_series else PFX_MID
                sys.stdout.write(prefix + series_template.format(**series))
