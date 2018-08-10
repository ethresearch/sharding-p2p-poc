#!/usr/bin/env python3

import json
import re
import os

URL_DELIMETER = '/'

gx_import_regex = "gx/ipfs/.+/.+"
general_import_regex = ".+/.+/.+"
quoted_gx_pkg_regex = f'"{gx_import_regex}"'

unwrite_pkgs = (
    "github.com/multiformats/go-multicodec/protobuf",
    "github.com/gogo/protobuf/proto",
)


def join_url(args):
    return URL_DELIMETER.join(args).rstrip(URL_DELIMETER)


class ProcessFailure(Exception):
    pass


class WrongFormat(Exception):
    pass


class ImportStatement:
    regex_pattern = None

    repo_name = None
    package_path = None

    @classmethod
    def is_import_format(cls, statement):
        return (
            statement == statement.strip(URL_DELIMETER) and
            cls.regex_pattern.search(statement) is not None
        )


class GxImport(ImportStatement):
    gx_id = 'gx{}ipfs'.format(URL_DELIMETER)
    regex_pattern = re.compile(gx_import_regex)

    gx_hash = None

    def __init__(self, statement):
        if not self.is_import_format(statement):
            raise WrongFormat("wrong format: {}".format(statement))
        exploded = statement[len(self.gx_id) + 1:].split(URL_DELIMETER)
        self.gx_hash = exploded[0]
        self.repo_name = exploded[1]
        self.package_path = join_url(exploded[2:])

    @property
    def statement(self):
        return join_url([
            self.gx_id,
            self.gx_hash,
            self.repo_name,
            self.package_path,
        ])

    @property
    def repo_path(self):
        return join_url([
            self.gx_id,
            self.gx_hash,
            self.repo_name,
        ])


class GeneralImport(ImportStatement):
    '''e.g. github.com/abc/edf
    '''
    regex_pattern = re.compile(general_import_regex)

    hostname = None
    username = None

    def __init__(self, statement):
        if not self.is_import_format(statement):
            raise WrongFormat("wrong format: {}".format(statement))
        exploded = statement.split(URL_DELIMETER)
        self.hostname = exploded[0]
        self.username = exploded[1]
        self.repo_name = exploded[2]
        self.package_path = join_url(exploded[3:])

    @property
    def statement(self):
        return join_url([
            self.hostname,
            self.username,
            self.repo_name,
            self.package_path,
        ])

    @property
    def repo_path(self):
        return join_url([
            self.hostname,
            self.username,
            self.repo_name,
        ])


def test_gx_import():
    s = "gx/ipfs/QmRDePEiL4Yupq5EkcK3L3ko3iMgYaqUdLu7xc1kqs7dnV/go-multicodec/protobuf"
    s_gx = GxImport(s)
    assert s_gx.gx_hash == "QmRDePEiL4Yupq5EkcK3L3ko3iMgYaqUdLu7xc1kqs7dnV"
    assert s_gx.repo_name == "go-multicodec"
    assert s_gx.package_path == "protobuf"
    assert s_gx.repo_path == "gx/ipfs/QmRDePEiL4Yupq5EkcK3L3ko3iMgYaqUdLu7xc1kqs7dnV/go-multicodec"
    assert s_gx.statement == s

    s2 = "gx/ipfs/QmRDePEiL4Yupq5EkcK3L3ko3iMgYaqUdLu7xc1kqs7dnV/go-multicodec/protobuf/"
    try:
        GxImport(s2)
        assert False
    except WrongFormat:
        pass

    s3 = "gx/ipfs/QmRDePEiL4Yupq5EkcK3L3ko3iMgYaqUdLu7xc1kqs7dnV"
    try:
        GxImport(s3)
        assert False
    except WrongFormat:
        pass


def test_general_import():
    s = "github.com/multiformats/go-multicodec"
    s_general = GeneralImport(s)
    assert s_general.hostname == "github.com"
    assert s_general.username == "multiformats"
    assert s_general.repo_name == "go-multicodec"
    assert s_general.repo_path == "github.com/multiformats/go-multicodec"
    assert s_general.statement == s

    s2 = "github.com/multiformats/go-multicodec/protobuf"
    s2_general = GeneralImport(s2)
    assert s2_general.hostname == "github.com"
    assert s2_general.username == "multiformats"
    assert s2_general.repo_name == "go-multicodec"
    assert s2_general.repo_path == "github.com/multiformats/go-multicodec"
    assert s2_general.package_path == "protobuf"
    assert s2_general.statement == s2

    s3 = "github.com/multiformats/go-multicodec/protobuf/"
    try:
        GeneralImport(s3)
        assert False
    except WrongFormat:
        pass


class GxImportConverter:
    go_path = None
    unwrite_pkgs = None
    cache = None

    def __init__(self, unwrite_pkgs):
        self.go_path = os.environ["GOPATH"]
        self.unwrite_pkgs = unwrite_pkgs
        self.cache = {}

    def run(self, gx_import_str):
        gx_import = GxImport(gx_import_str)
        gx_import_repo_path = join_url([
            self.go_path,
            "src",
            gx_import.repo_path,
        ])
        json_file_path = join_url([
            gx_import_repo_path,
            "package.json",
        ])
        with open(json_file_path, 'r') as f_read:
            package_info = json.load(f_read)
            orig_repo_path = package_info['gx']['dvcsimport']
            return join_url([
                orig_repo_path,
                gx_import.package_path,
            ])

    def convert(self, gx_import_str):
        if gx_import_str in self.cache:
            return self.cache[gx_import_str]
        result = self.run(gx_import_str)
        self.cache[gx_import_str] = result
        if result not in self.unwrite_pkgs:
            return gx_import_str
        return result


def test_gx_import_converter():
    temp_unwrite_pkgs = []
    converter = GxImportConverter(temp_unwrite_pkgs)
    s_gx = "gx/ipfs/QmRDePEiL4Yupq5EkcK3L3ko3iMgYaqUdLu7xc1kqs7dnV/go-multicodec/protobuf"
    s_general = "github.com/multiformats/go-multicodec/protobuf"
    assert converter.convert(s_gx) == s_gx
    converter.unwrite_pkgs.append(s_general)
    assert converter.convert(s_gx) == s_general


class Processor:

    def preprocess(self, string):
        if len(string) < 2:
            raise ProcessFailure("len of {} < 2".format(string))
        if string[0] != '"' or string[0] != '"':
            raise ProcessFailure("wrong format: {}".format(string))
        return string[1:-1]

    def postprocess(self, string):
        return f"\"{string}\""


def test_processor():
    p = Processor()
    try:
        p.preprocess("123")
        assert False
    except:
        pass
    assert p.preprocess('"123"') == '123'
    assert p.postprocess('123') == '"123"'


class LineConverter:
    import_converter = None
    processor = None
    pattern = None

    def __init__(self, regex_str, processor, import_converter):
        self.import_converter = import_converter
        self.processor = processor
        self.pattern = re.compile(regex_str)

    def convert(self, line):
        m = self.pattern.search(line)
        if m is None:
            return line
        matched_str = m.group(0)
        preprocessed_str = self.processor.preprocess(matched_str)
        converted_str = self.import_converter.convert(preprocessed_str)
        postprocessed_str = self.processor.postprocess(converted_str)
        return line[:m.start()] + postprocessed_str + line[m.end():]


def test_line_converter():
    temp_unwrite_pkgs = unwrite_pkgs
    line_converter = LineConverter(
        quoted_gx_pkg_regex,
        Processor(),
        GxImportConverter(temp_unwrite_pkgs),
    )
    line = '    protobufCodec "gx/ipfs/QmRDePEiL4Yupq5EkcK3L3ko3iMgYaqUdLu7xc1kqs7dnV/go-multicodec/protobuf"'
    assert line_converter.convert(line) == '    protobufCodec "github.com/multiformats/go-multicodec/protobuf"'
    line_not_matched = '    protobufCodec "1gx/ipfs/QmRDePEiL4Yupq5EkcK3L3ko3iMgYaqUdLu7xc1kqs7dnV/go-multicodec/protobuf"'
    assert line_converter.convert(line_not_matched) == line_not_matched


def all_go_files(root_path):
    go_ext = '.go'
    for path, _, files in os.walk(root_path):
        for filename in files:
            if filename[-len(go_ext):] == go_ext:
                file_path = "{}/{}".format(path, filename)
                yield file_path


class FileUnwriter:
    temp_unwritten_file_fmt = '.{}.uwbk'

    line_unwriter = None

    def __init__(self, line_unwriter):
        self.line_unwriter = line_unwriter

    def unwrite(self, file_path):
        temp_unwritten_file = "{}/{}".format(
            os.path.dirname(file_path),
            self.temp_unwritten_file_fmt.format(os.path.basename(file_path)),
        )
        with open(file_path, 'r') as f_read, open(temp_unwritten_file, 'w') as f_write:
            for line in f_read.readlines():
                unwritten_line = self.line_unwriter.convert(line)
                f_write.write(unwritten_line)
        os.rename(temp_unwritten_file, file_path)


def test_file_unwriter():
    temp_unwrite_pkgs = (
        "github.com/gogo/protobuf/proto",
        "github.com/multiformats/go-multicodec/protobuf",
    )
    content = """
    protobufCodec "gx/ipfs/QmRDePEiL4Yupq5EkcK3L3ko3iMgYaqUdLu7xc1kqs7dnV/go-multicodec/protobuf"
    "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
"""
    line_unwriter = LineConverter(
        quoted_gx_pkg_regex,
        Processor(),
        GxImportConverter(temp_unwrite_pkgs),
    )
    file_unwriter = FileUnwriter(line_unwriter)

    test_file = "./temppppppppppppppppppppppp.txt"
    with open(test_file, 'w') as f_write:
        f_write.write(content)
    file_unwriter.unwrite(test_file)
    try:
        f_read = open(test_file, 'r')
        read_content = f_read.read()
    except IOError:
        assert False
    expected_content = """
    protobufCodec "github.com/multiformats/go-multicodec/protobuf"
    "github.com/gogo/protobuf/proto"
"""
    assert read_content == expected_content
    os.unlink(test_file)

    # for file_path in all_go_files('.'):
    #     file_unwriter.unwrite(file_path)


def test():
    # import_map = (
    #     ("github.com/multiformats/go-multicodec/protobuf", "gx/ipfs/QmRDePEiL4Yupq5EkcK3L3ko3iMgYaqUdLu7xc1kqs7dnV/go-multicodec/protobuf"),
    #     ("github.com/gogo/protobuf/proto", "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"),
    # )
    test_gx_import()
    test_general_import()
    test_gx_import_converter()
    test_processor()
    test_line_converter()
    test_file_unwriter()


if __name__ == '__main__':
    main()
