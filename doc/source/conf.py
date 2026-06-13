# Configuration file for the Sphinx documentation builder.
#
# For the full list of built-in configuration values, see the documentation:
# https://www.sphinx-doc.org/en/master/usage/configuration.html

# -- Project information -----------------------------------------------------
# https://www.sphinx-doc.org/en/master/usage/configuration.html#project-information

import requests

project = 'blunderDB'
copyright = '2024-2026, Kevin UNGER <blunderdb@proton.me>'
author = 'Kevin UNGER <blunderdb@proton.me>'
release = '0.27.0'

# -- General configuration ---------------------------------------------------
# https://www.sphinx-doc.org/en/master/usage/configuration.html#general-configuration

extensions = [
        'sphinx_rtd_theme',
        'sphinxcontrib.youtube'
        ]

# Documentation languages. French is the *source* language (the .rst files are
# written in French); every other language is a gettext translation living under
# locale/<code>/LC_MESSAGES/*.po. This single list drives the build (doc/build.py
# loops over the codes), the in-page language switcher (html_context below) and
# the per-language PDF download links (rst_prolog below). Add a language here and
# the whole pipeline picks it up.
LANGUAGES = [
        ("fr", "Français"),
        ("en", "English"),
        ("de", "Deutsch"),
        ("el", "Ελληνικά"),
        ("es", "Español"),
        ("fi", "Suomi"),
        ("it", "Italiano"),
        ("ja", "日本語"),
        ("ru", "Русский"),
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
# The language switcher (see _templates/versions.html) renders one link per
# entry. Each entry is [code, name, url]: the short code (e.g. "fr") is the
# visible label so the switcher stays compact and never overlaps the nav, the
# native name (e.g. "Français") is the hover tooltip, and url is a sibling
# directory produced by doc/build.py.
html_context = {
        'languages': [[code, name, "../" + code] for code, name in LANGUAGES]
        }

# Construct the latest Windows executable URL
_releases = "https://github.com/kevung/blunderDB/releases"
if release:
    _dl = f"{_releases}/latest/download"
    latest_windows_exe_url = f"{_dl}/blunderDB-windows-{release}.exe"
    latest_linux_exe_url = f"{_dl}/blunderDB-linux-{release}"
    latest_linux_webkit2gtk41_exe_url = f"{_dl}/blunderDB-linux-webkit2gtk-4.1-{release}"
    latest_mac_exe_url = f"{_dl}/blunderDB-macos-{release}.zip"
    # One PDF per documentation language: blunderDB-<release>-<code>.pdf
    latest_pdf_urls = {code: f"{_dl}/blunderDB-{release}-{code}.pdf"
                       for code, _ in LANGUAGES}
else:
    latest_windows_exe_url = _releases  # Fallback URL
    latest_linux_exe_url = _releases  # Fallback URL
    latest_linux_webkit2gtk41_exe_url = _releases  # Fallback URL
    latest_mac_exe_url = _releases  # Fallback URL
    latest_pdf_urls = {code: _releases for code, _ in LANGUAGES}

# Per-language PDF substitutions: |latest_fr_pdf|, |latest_en_pdf|, … one for
# each documentation language so any page can link its own (or every) PDF.
_pdf_substitutions = "\n".join(
        f".. |latest_{code}_pdf| replace:: `{url} <{url}>`__"
        for code, url in latest_pdf_urls.items())

# Add it as a Sphinx variable
rst_prolog = f"""
.. |latest_windows_exe| replace:: `{latest_windows_exe_url} <{latest_windows_exe_url}>`__
.. |latest_linux_exe| replace:: `{latest_linux_exe_url} <{latest_linux_exe_url}>`__
.. |latest_linux_webkit2gtk41_exe| replace:: `{latest_linux_webkit2gtk41_exe_url} <{latest_linux_webkit2gtk41_exe_url}>`__
.. |latest_mac_exe| replace:: `{latest_mac_exe_url} <{latest_mac_exe_url}>`__
{_pdf_substitutions}
"""

# -- Options for LaTeX / PDF output ------------------------------------------
# Build PDFs with XeLaTeX. The default engine (pdflatex) aborts with a fatal
# "Unicode character not set up for use with LaTeX" error on the symbols used
# in the French docs (↔ → ≤ ×, …); XeLaTeX handles arbitrary Unicode natively.
# The CI already installs texlive-xetex for this.
latex_engine = 'xelatex'

# Per-language font/engine setup. Sphinx's -D language=<code> override is not
# visible while conf.py runs, so doc/build.py passes the target language via the
# BLUNDERDB_DOC_LANG environment variable. Latin-script languages (fr, en, de,
# es, fi, it) keep XeLaTeX with the default font setup that already produces the
# FR/EN PDFs.
import os
_doc_lang = os.environ.get('BLUNDERDB_DOC_LANG', language)
latex_elements = {}
if _doc_lang == 'ja':
    # Japanese: XeLaTeX is incompatible with Sphinx's `japanese` document-class
    # option (it switches sphinxmanual into pLaTeX2e mode). Use Sphinx's
    # supported Japanese toolchain instead — upLaTeX + dvipdfmx — which renders
    # CJK natively via the default kanji fonts (HaranoAji, shipped with
    # texlive-lang-cjk / texlive-lang-japanese). No xeCJK preamble needed.
    latex_engine = 'uplatex'
elif _doc_lang in ('el', 'ru'):
    # Greek and Cyrillic under XeLaTeX: pin GNU FreeFont (fonts-freefont-otf),
    # which covers both scripts, in case the default main font does not.
    latex_elements['fontpkg'] = r'''
\setmainfont{FreeSerif}
\setsansfont{FreeSans}
\setmonofont{FreeMono}
'''

