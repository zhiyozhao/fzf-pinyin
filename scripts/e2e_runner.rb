# frozen_string_literal: true

require 'json'
require 'timeout'
require 'minitest/autorun'
require_relative 'e2e_processes'

module Minitest
  def self.plugin_fzfp_results_init(_options)
    reporter << E2EReporter.new
  end

  class E2EReporter < AbstractReporter
    def record(result)
      status = if result.skipped? then 'skip'
               elsif result.passed? then 'pass'
               elsif result.error? then 'error'
               else 'fail'
               end
      row = { id: "#{result.klass}##{result.name}", status: status, seconds: result.time,
              assertions: result.assertions, failures: result.failures.map { { type: it.class.name, message: it.message, backtrace: it.backtrace } } }
      File.open(ENV.fetch('E2E_RESULTS'), 'a') { it.puts(JSON.generate(row)) }
    end
  end
  extensions << 'fzfp_results'
end

# Fail with a recorded error instead of hanging indefinitely on a FIFO read.
# The optional exhaustive layout matrix has its own much larger allowance.
module E2ETimeout
  def capture_exceptions(&block)
    super { Timeout.timeout(name == 'test_combinations' ? 3600 : 120, &block) }
  end
end
Minitest::Test.prepend(E2ETimeout)

suite = ARGV.shift
Dir[File.join(suite, 'test_*.rb')].sort.each do |path|
  require path unless File.basename(path) == 'test_shell_integration.rb'
end

# Snapshot pane descendants before the upstream teardown closes the window,
# while ancestry still identifies them; reap lingering preview/reload commands.
module E2EPaneCleanup
  def kill
    roots = IO.popen(['tmux', 'list-panes', '-t', win, '-F', '#{pane_pid}'], err: File::NULL, &:read)
              .lines.map(&:to_i).select(&:positive?)
    owned = E2EProcesses.descendants(roots)
    super
  ensure
    E2EProcesses.stop(owned || [])
  end
end
Tmux.prepend(E2EPaneCleanup)

manifest = Minitest::Runnable.runnables.flat_map do |klass|
  klass.public_instance_methods(true).grep(/^test_/).map { "#{klass.name}##{it}" }
end.sort
File.write(ENV.fetch('E2E_MANIFEST'), JSON.generate(manifest))
