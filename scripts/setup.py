from setuptools import setup, find_packages
from pathlib import Path

root = Path()

setup(
    name="oiaj-scripts",
    version="1.0",
    package_dir={'': 'src'},
    packages=find_packages(where='src'),
    package_data={
        'oiaj-scripts': [
            '../README.md',
        ],
    },
    entry_points={
        'console_scripts': [
            'oia=oia.main:main',
        ],
    },
    python_requires='>=3',
    install_requires=[],
)
