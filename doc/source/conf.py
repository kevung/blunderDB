# Configuration file for the Sphinx documentation builder.
#
# For the full list of built-in configuration values, see the documentation:
# https://www.sphinx-doc.org/en/master/usage/configuration.html

# -- Project information -----------------------------------------------------
# https://www.sphinx-doc.org/en/master/usage/configuration.html#project-information

import requests

project = 'blunderDB'
copyright = '2024, Kevin UNGER <blunderdb@proton.me>'
author = 'Kevin UNGER <blunderdb@proton.me>'
release = '0.9.0'

# -- General configuration ---------------------------------------------------
# https://www.sphinx-doc.org/en/master/usage/configuration.html#general-configuration

extensions = [
        'sphinx_rtd_theme',
        'sphinxcontrib.youtube'
        ]

language = 'fr'
templates_path = ['_templates']
locale_dirs = ['locale/']
gettext_compact = False
exclude_patterns = []
html_theme = "sphinx_rtd_theme"
html_static_path = ['_static']
html_css_files = [
        'theme_overrides.css',
        'custom.css',
         ]
html_show_sphinx = False
html_show_sourcelink = False
html_favicon = '_static/favicon.ico'
html_logo = '_static/logo.png'
html_context = {
        'languages': [["en", "../en"], ["fr", "../fr"]]
        }

# Construct the latest Windows executable URL
if release:
    latest_windows_exe_url = f"https://github.com/kevung/blunderDB/releases/latest/download/blunderDB-windows-{release}.exe"
    latest_linux_exe_url = f"https://github.com/kevung/blunderDB/releases/latest/download/blunderDB-linux-{release}"
    latest_mac_exe_url = f"https://github.com/kevung/blunderDB/releases/latest/download/blunderDB-macos-{release}.zip"
    latest_fr_pdf_url = f"https://github.com/kevung/blunderDB/releases/latest/download/blunderDB-{release}-fr.pdf"
    latest_en_pdf_url = f"https://github.com/kevung/blunderDB/releases/latest/download/blunderDB-{release}-en.pdf"
    
else:
    latest_windows_exe_url = "https://github.com/kevung/blunderDB/releases"  # Fallback URL
    latest_linux_exe_url = "https://github.com/kevung/blunderDB/releases"  # Fallback URL
    latest_mac_exe_url = "https://github.com/kevung/blunderDB/releases"  # Fallback URL
    latest_fr_pdf_url = "https://github.com/kevung/blunderDB/releases"  # Fallback URL
    latest_en_pdf_url = "https://github.com/kevung/blunderDB/releases"  # Fallback URL
    

# Add it as a Sphinx variable
rst_prolog = f"""
.. |latest_windows_exe| replace:: `{latest_windows_exe_url} <{latest_windows_exe_url}>`__
.. |latest_linux_exe| replace:: `{latest_linux_exe_url} <{latest_linux_exe_url}>`__
.. |latest_fr_pdf| replace:: `{latest_fr_pdf_url} <{latest_fr_pdf_url}>`__
.. |latest_en_pdf| replace:: `{latest_en_pdf_url} <{latest_en_pdf_url}>`__
.. |latest_mac_exe| replace:: `{latest_mac_exe_url} <{latest_mac_exe_url}>`__
"""

