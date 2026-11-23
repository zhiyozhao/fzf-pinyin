# frozen_string_literal: true

module E2EProcesses
  module_function

  def descendants(roots)
    # Process identities only, never other processes' arguments/environment.
    pairs = IO.popen(['/bin/ps', '-axo', 'pid=,ppid='], &:read).lines.map { it.split.map(&:to_i) }
    found = roots.compact
    loop do
      more = pairs.filter_map { |pid, parent| pid if found.include?(parent) && !found.include?(pid) }
      break if more.empty?

      found += more
    end
    found
  end

  def stop(pids)
    pids.reverse_each do |pid|
      Process.kill('TERM', pid)
    rescue Errno::ESRCH
      nil
    end
    deadline = Process.clock_gettime(Process::CLOCK_MONOTONIC) + 2
    loop do
      pids.select! do |pid|
        Process.kill(0, pid)
        true
      rescue Errno::ESRCH
        false
      end
      break if pids.empty? || Process.clock_gettime(Process::CLOCK_MONOTONIC) > deadline

      sleep 0.05
    end
    pids.each do |pid|
      Process.kill('KILL', pid)
    rescue Errno::ESRCH
      nil
    end
  end
end
