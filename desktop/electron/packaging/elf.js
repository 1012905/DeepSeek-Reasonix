"use strict";
const fs = require("node:fs");

// Only the Electron shell may carry a glibc floor. A Go binary that names a
// program interpreter inherits the build runner's glibc and will not start on
// an older system, so the Linux package refuses one.
const PT_INTERP = 3;

function elfInterpreter(bytes) {
  if (bytes.length < 52 || bytes.readUInt32BE(0) !== 0x7f454c46) throw new Error("not an ELF file");
  const wide = bytes[4] === 2;
  const little = bytes[5] === 1;
  if (bytes[4] !== 1 && !wide) throw new Error(`unknown ELF class ${bytes[4]}`);
  if (bytes[5] !== 2 && !little) throw new Error(`unknown ELF data encoding ${bytes[5]}`);
  if (wide && bytes.length < 64) throw new Error("truncated ELF header");
  const u16 = (at) => (little ? bytes.readUInt16LE(at) : bytes.readUInt16BE(at));
  const u32 = (at) => (little ? bytes.readUInt32LE(at) : bytes.readUInt32BE(at));
  const word = (at) => (wide ? Number(little ? bytes.readBigUInt64LE(at) : bytes.readBigUInt64BE(at)) : u32(at));
  const phoff = word(wide ? 32 : 28);
  const phentsize = u16(wide ? 54 : 42);
  const phnum = u16(wide ? 56 : 44);
  for (let i = 0; i < phnum; i++) {
    const header = phoff + i * phentsize;
    if (header + phentsize > bytes.length) throw new Error("truncated ELF program header table");
    if (u32(header) !== PT_INTERP) continue;
    const offset = word(header + (wide ? 8 : 4));
    const size = word(header + (wide ? 32 : 16));
    if (offset + size > bytes.length) throw new Error("truncated ELF interpreter path");
    return bytes.toString("latin1", offset, offset + size).replace(/\0+$/, "");
  }
  return null;
}

function dynamicallyLinked(files) {
  const found = [];
  for (const file of files) {
    const interpreter = elfInterpreter(fs.readFileSync(file));
    if (interpreter !== null) found.push({ file, interpreter });
  }
  return found;
}

// e_machine, by the GOARCH it answers to. A Go binary of the other
// architecture still starts under emulation or a compat layer on some systems, so
// "it runs" says nothing about which one a package carries.
const MACHINES = { amd64: 62, arm64: 183 };

function elfMachine(bytes) {
  if (bytes.length < 20 || bytes.readUInt32BE(0) !== 0x7f454c46) throw new Error("not an ELF file");
  const code = bytes[5] === 1 ? bytes.readUInt16LE(18) : bytes.readUInt16BE(18);
  return Object.keys(MACHINES).find((arch) => MACHINES[arch] === code) ?? `machine ${code}`;
}

function wrongArchitecture(files, arch) {
  if (!MACHINES[arch]) throw new Error(`no ELF machine is known for ${arch}`);
  const found = [];
  for (const file of files) {
    const machine = elfMachine(fs.readFileSync(file));
    if (machine !== arch) found.push({ file, machine });
  }
  return found;
}

module.exports = { elfInterpreter, dynamicallyLinked, elfMachine, wrongArchitecture };

if (require.main === module) {
  const args = process.argv.slice(2);
  let arch = "";
  if (args[0] === "--arch") {
    arch = args[1] ?? "";
    args.splice(0, 2);
  }
  if (args.length === 0) {
    console.error("usage: node packaging/elf.js [--arch <goarch>] <go-binary>...");
    process.exit(2);
  }
  const found = dynamicallyLinked(args);
  for (const { file, interpreter } of found) {
    console.error(`${file} is dynamically linked (interpreter ${interpreter}); build it with CGO_ENABLED=0`);
  }
  const wrong = arch ? wrongArchitecture(args, arch) : [];
  for (const { file, machine } of wrong) {
    console.error(`${file} is built for ${machine}, but the package is named for ${arch}`);
  }
  process.exit(found.length === 0 && wrong.length === 0 ? 0 : 1);
}
