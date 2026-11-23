# frozen_string_literal: true

# Run the same pinned upstream assertions against both executables. Only the
# fixture's dependency/bootstrap, temporary paths and listen port are adapted.
require 'fileutils'
require 'digest'
require 'json'
require 'open3'
require 'rbconfig'
require 'shellwords'
require 'socket'
require 'tmpdir'
require_relative 'e2e_processes'
gem 'minitest', '= 5.25.4'
require 'minitest'
$stdout.sync = true

ROOT = File.expand_path('..', __dir__)
UPSTREAM = 'bf4869b7fa7748d973eeb414da4dde0d7fe4cd3c'
OUT = File.expand_path(ENV.fetch('E2E_OUTPUT', 'Artifacts/e2e'), ROOT)
REAL_TMUX = ENV.fetch('E2E_TMUX', '/opt/homebrew/bin/tmux')
SAFE_PATH = '/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin'

def run_suite(label, binary)
  started = Process.clock_gettime(Process::CLOCK_MONOTONIC)
  temp = Dir.mktmpdir('fzfp-e2e-', '/tmp')
  fixture = File.join(temp, 'fixture')
  home = File.join(temp, 'home')
  shim = File.join(temp, 'shims')
  scratch = File.join(temp, 'tmp')
  socket = File.join(temp, 'tmux.sock')
  FileUtils.mkdir_p([fixture, home, shim, scratch])
  env = {
    'HOME' => home, 'TMPDIR' => scratch, 'XDG_CONFIG_HOME' => "#{home}/config",
    'XDG_DATA_HOME' => "#{home}/data", 'XDG_CACHE_HOME' => "#{home}/cache",
    'HISTFILE' => "#{home}/history", 'INPUTRC' => '/dev/null', 'SHELL' => '/bin/bash',
    'PATH' => "#{shim}:#{fixture}/bin:#{SAFE_PATH}", 'TERM' => 'xterm-256color',
    'LANG' => 'en_US.UTF-8', 'LC_ALL' => 'en_US.UTF-8',
    'GEM_HOME' => "#{ROOT}/.cache/gems", 'GEM_PATH' => "#{ROOT}/.cache/gems",
    'MT_CPU' => '1', 'E2E_RESULTS' => "#{OUT}/#{label}.jsonl",
    'E2E_MANIFEST' => "#{OUT}/#{label}-manifest.json",
    'LONGTEST' => ENV.fetch('E2E_LONGTEST', '1') == '1' ? '1' : nil
  }
  # Archive tracked upstream data only: never copy .git, caches or user data.
  Open3.pipeline(['git', '-C', ROOT, 'archive', UPSTREAM], ['tar', '-xf', '-', '-C', fixture]).each do |status|
    raise 'Could not extract upstream fixtures' unless status.success?
  end
  FileUtils.ln_s(binary, "#{fixture}/bin/fzf")
  FileUtils.ln_s(RbConfig.ruby, "#{shim}/ruby")
  File.write("#{shim}/tmux", "#!/bin/sh\nexec #{REAL_TMUX.shellescape} -S #{socket.shellescape} -f /dev/null \"$@\"\n")
  FileUtils.chmod(0o755, "#{shim}/tmux")
  # Keep assertions and test names intact; relocate unsafe global fixture paths.
  Dir["#{fixture}/test/**/*.{rb,sh,fish}"].each do |path|
    source = File.read(path).gsub('/tmp/', "#{scratch}/")
    source = source.sub("require 'bundler/setup'", "# Gems are selected by the isolated GEM_HOME/GEM_PATH.")
    File.write(path, source)
  end
  # Avoid the upstream fixed port 6266, which might belong to another process.
  port = TCPServer.open('127.0.0.1', 0) { it.addr[1] }
  server_test = "#{fixture}/test/test_server.rb"
  File.write(server_test, File.read(server_test).gsub('6266', port.to_s))
  runner = File.join(ROOT, 'scripts/e2e_runner.rb')
  log = File.open("#{OUT}/#{label}.log", 'w')
  File.write(env.fetch('E2E_RESULTS'), '')
  tmux = [REAL_TMUX, '-S', socket, '-f', '/dev/null']
  system(env, *tmux, 'new-session', '-d', '-x', '80', '-y', '24', '-s', 'e2e',
         '/bin/bash --noprofile --norc', unsetenv_others: true, chdir: fixture, out: log, err: log) or raise 'tmux startup failed'
  server_pid = IO.popen(env, tmux + ['display-message', '-p', '#{pid}'], unsetenv_others: true, &:read).to_i
  raise 'Missing owned tmux server PID' unless server_pid.positive?

  pid = Process.spawn(env, RbConfig.ruby, runner, "#{fixture}/test", '--seed', '20261123',
                      unsetenv_others: true, chdir: fixture, pgroup: true, out: log, err: log)
  puts "#{label}: running all upstream suites (log: #{OUT}/#{label}.log)"
  status = Process.wait2(pid).last
  pid = nil # Reaped: do not signal a PID that the OS may subsequently reuse.
  rows = File.readlines(env.fetch('E2E_RESULTS')).map { JSON.parse(it) }
  raise "#{label}: no test results (see log)" if rows.empty?
  expected = JSON.parse(File.read(env.fetch('E2E_MANIFEST')))
  actual = rows.map { it.fetch('id') }.sort
  raise "#{label}: incomplete or duplicate suite results" unless actual == expected

  counts = %w[pass fail error skip].to_h { [it, 0] }.merge(rows.group_by { it.fetch('status') }.transform_values(&:length))
  summary = { counts: counts, tests: rows.length, seconds: (Process.clock_gettime(Process::CLOCK_MONOTONIC) - started).round(2),
              exit_code: status.exitstatus, ruby: RUBY_DESCRIPTION,
              binary_sha256: Digest::SHA256.file(binary).hexdigest,
              version: Open3.capture3(env, binary, '--version', unsetenv_others: true).first.strip,
              suites: rows.group_by { it.fetch('id').split('#').first }.transform_values(&:length) }
  File.write("#{OUT}/#{label}.json", JSON.pretty_generate(summary) + "\n")
  puts "#{label}: #{summary.to_json}"
  [rows, summary]
ensure
  if socket && File.socket?(socket) && !server_pid&.positive?
    server_pid = IO.popen(env, tmux + ['display-message', '-p', '#{pid}'], unsetenv_others: true, err: File::NULL, &:read).to_i
  end
  owned = E2EProcesses.descendants([pid, server_pid].compact.select(&:positive?))
  system(env, *tmux, 'kill-server', unsetenv_others: true, out: File::NULL, err: File::NULL) if socket && File.socket?(socket)
  E2EProcesses.stop(owned)
  log&.close
  FileUtils.remove_entry(temp) if temp && File.directory?(temp)
end

FileUtils.mkdir_p(OUT)
%w[INT TERM HUP].each { |signal| Signal.trap(signal) { raise Interrupt, signal } }
baseline, baseline_summary = run_suite('upstream', File.expand_path(ENV.fetch('FZF_REFERENCE_BIN', 'Artifacts/fzf-reference'), ROOT))
candidate, candidate_summary = run_suite('fzfp', File.expand_path(ENV.fetch('FZFP_BIN', 'Artifacts/fzfp'), ROOT))
baseline_by_id = baseline.to_h { [it.fetch('id'), it] }
candidate_by_id = candidate.to_h { [it.fetch('id'), it] }
missing = baseline_by_id.keys - candidate_by_id.keys
extra = candidate_by_id.keys - baseline_by_id.keys
regressions = candidate.select do |row|
  %w[fail error].include?(row.fetch('status')) && !%w[fail error].include?(baseline_by_id.dig(row.fetch('id'), 'status'))
end
new_skips = candidate.select { it['status'] == 'skip' && baseline_by_id.dig(it['id'], 'status') != 'skip' }
comparison = { upstream: baseline_summary, fzfp: candidate_summary, missing: missing, extra: extra,
               new_failures: regressions, new_skips: new_skips,
               shared_failures: candidate.filter_map { |row| { upstream: baseline_by_id[row['id']], fzfp: row } if %w[fail error].include?(row['status']) && !regressions.include?(row) },
               status_changes: candidate.filter_map { |row| { id: row['id'], upstream: baseline_by_id.dig(row['id'], 'status'), fzfp: row['status'] } if baseline_by_id.dig(row['id'], 'status') != row['status'] } }
File.write("#{OUT}/comparison.json", JSON.pretty_generate(comparison) + "\n")
puts "Comparison: #{regressions.length} new failures, #{new_skips.length} new skips, #{missing.length} missing, #{extra.length} extra (#{OUT}/comparison.json)"
exit(regressions.empty? && new_skips.empty? && missing.empty? && extra.empty? ? 0 : 1)
