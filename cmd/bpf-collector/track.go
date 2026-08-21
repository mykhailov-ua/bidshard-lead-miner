package main

import (
	"fmt"
	"log/slog"
)

func (r *probeRun) trackPID(pid uint32, role uint8, name string) error {
	return r.trackTarget(pid, 0, role, name)
}

func (r *probeRun) trackTarget(pid uint32, cgroupID uint64, role uint8, name string) error {
	if pid == 0 && cgroupID == 0 {
		return fmt.Errorf("invalid target")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if pid != 0 {
		if existing, ok := r.tracked[pid]; ok {
			// Duplicate track is a no-op; refresh cgroup map when cgroup_id changes for same PID.
			if cgroupID != 0 && cgroupID != existing.CgroupID {
				_ = r.coll.PutTargetCgroup(cgroupID, role)
			}
			return nil
		}
		if err := r.coll.PutTargetPID(pid, role); err != nil {
			return err
		}
	}
	if cgroupID != 0 {
		if err := r.coll.PutTargetCgroup(cgroupID, role); err != nil {
			return err
		}
	}
	if pid != 0 {
		r.tracked[pid] = targetEntry{PID: pid, CgroupID: cgroupID, Role: role, Name: name}
		slog.Info("tracking pid", "pid", pid, "cgroup_id", cgroupID, "role", roleName(role), "name", name)
	} else if cgroupID != 0 {
		slog.Info("tracking cgroup", "cgroup_id", cgroupID, "role", roleName(role), "name", name)
	}
	return nil
}

func roleName(role uint8) string {
	switch role {
	case roleParser:
		return "parser"
	case roleTelegram:
		return "telegram"
	case roleMongo:
		return "mongo"
	case roleLoadgen:
		return "loadgen"
	case roleWorker:
		return "worker"
	default:
		return "unknown"
	}
}
