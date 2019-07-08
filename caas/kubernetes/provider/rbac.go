// Copyright 2019 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package provider

import (
	"github.com/juju/errors"
	core "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/juju/juju/caas"
)

func (k *kubernetesClient) getRBACLabels(appName string) map[string]string {
	return map[string]string{
		labelApplication: appName,
		labelModel:       k.namespace,
	}
}

func (k *kubernetesClient) ensureServiceAccountForApp(
	appName string, caasSpec *caas.ServiceAccountSpec,
) (cleanups []func(), err error) {
	labels := k.getRBACLabels(appName)
	saSpec := &core.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Name:      caasSpec.Name,
			Namespace: k.namespace,
			Labels:    labels,
		},
		AutomountServiceAccountToken: caasSpec.AutomountServiceAccountToken,
	}
	// ensure service account;
	sa, err := k.updateServiceAccountForApp(appName, saSpec)
	if err != nil {
		if !errors.IsNotFound(err) {
			return cleanups, errors.Trace(err)
		}
		if sa, err = k.createServiceAccount(saSpec); err != nil {
			return cleanups, errors.Trace(err)
		}
		cleanups = append(cleanups, func() { k.deleteServiceAccount(sa.GetName()) })
	}

	roleSpec := caasSpec.Capabilities.Role
	var roleRef rbacv1.RoleRef
	switch roleSpec.Type {
	case caas.Role:
		var r *rbacv1.Role
		if len(roleSpec.Rules) == 0 {
			// no rules was specified, reference to an existing Role.
			r, err = k.getRole(roleSpec.Name)
			if err != nil {
				return cleanups, errors.Trace(err)
			}
		} else {
			// create or update Role.
			r, err = k.ensureRole(&rbacv1.Role{
				ObjectMeta: v1.ObjectMeta{
					Name:      roleSpec.Name,
					Namespace: k.namespace,
					Labels:    labels,
				},
				Rules: roleSpec.Rules,
			})
			if err != nil {
				return cleanups, errors.Trace(err)
			}
			cleanups = append(cleanups, func() { k.deleteRole(r.GetName()) })
		}
		roleRef = rbacv1.RoleRef{
			Name: r.GetName(),
			Kind: string(roleSpec.Type),
		}
	case caas.ClusterRole:
		var cr *rbacv1.ClusterRole
		if len(roleSpec.Rules) == 0 {
			// no rules was specified, reference to an existing ClusterRole.
			cr, err = k.getClusterRole(roleSpec.Name)
			if err != nil {
				return cleanups, errors.Trace(err)
			}
		} else {
			// create or update ClusterRole.
			cr, err = k.ensureClusterRole(&rbacv1.ClusterRole{
				ObjectMeta: v1.ObjectMeta{
					Name:      roleSpec.Name,
					Namespace: k.namespace,
					Labels:    labels,
				},
				Rules: roleSpec.Rules,
			})
			if err != nil {
				return cleanups, errors.Trace(err)
			}
			cleanups = append(cleanups, func() { k.deleteClusterRole(cr.GetName()) })
		}
		roleRef = rbacv1.RoleRef{
			Name: cr.GetName(),
			Kind: string(roleSpec.Type),
		}
	default:
		// this should never happen.
		return cleanups, errors.New("unsupported Role type")
	}

	rbSpec := caasSpec.Capabilities.RoleBinding
	roleBindingMeta := v1.ObjectMeta{
		Name:      rbSpec.Name,
		Namespace: sa.GetNamespace(),
		Labels:    labels,
	}
	roleBindingSASubject := rbacv1.Subject{
		Kind:      rbacv1.ServiceAccountKind,
		Name:      sa.GetName(),
		Namespace: sa.GetNamespace(),
	}
	switch rbSpec.Type {
	case caas.RoleBinding:
		rb, err := k.ensureRoleBinding(&rbacv1.RoleBinding{
			ObjectMeta: roleBindingMeta,
			RoleRef:    roleRef,
			Subjects:   []rbacv1.Subject{roleBindingSASubject},
		})
		if err != nil {
			return cleanups, errors.Trace(err)
		}
		cleanups = append(cleanups, func() { k.deleteRoleBinding(rb.GetName()) })
	case caas.ClusterRoleBinding:
		crb, err := k.ensureClusterRoleBinding(&rbacv1.ClusterRoleBinding{
			ObjectMeta: roleBindingMeta,
			RoleRef:    roleRef,
			Subjects:   []rbacv1.Subject{roleBindingSASubject},
		})
		if err != nil {
			return cleanups, errors.Trace(err)
		}
		cleanups = append(cleanups, func() { k.deleteClusterRoleBinding(crb.GetName()) })
	default:
		// this should never happen.
		return cleanups, errors.New("unsupported Role binding type")
	}
	return cleanups, nil
}

func (k *kubernetesClient) createServiceAccount(sa *core.ServiceAccount) (*core.ServiceAccount, error) {
	out, err := k.client().CoreV1().ServiceAccounts(k.namespace).Create(sa)
	if k8serrors.IsAlreadyExists(err) {
		return nil, errors.AlreadyExistsf("service account %q", sa.GetName())
	}
	return out, errors.Trace(err)
}

func (k *kubernetesClient) updateServiceAccount(sa *core.ServiceAccount) (*core.ServiceAccount, error) {
	out, err := k.client().CoreV1().ServiceAccounts(k.namespace).Update(sa)
	if k8serrors.IsNotFound(err) {
		return nil, errors.NotFoundf("service account %q", sa.GetName())
	}
	return out, errors.Trace(err)
}

func (k *kubernetesClient) updateServiceAccountForApp(appName string, sa *core.ServiceAccount) (*core.ServiceAccount, error) {
	_, err := k.listServiceAccount(appName)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return k.updateServiceAccount(sa)
}

func (k *kubernetesClient) ensureServiceAccount(sa *core.ServiceAccount) (*core.ServiceAccount, error) {
	out, err := k.updateServiceAccount(sa)
	if errors.IsNotFound(err) {
		out, err = k.createServiceAccount(sa)
	}
	return out, errors.Trace(err)
}

func (k *kubernetesClient) getServiceAccount(name string) (*core.ServiceAccount, error) {
	out, err := k.client().CoreV1().ServiceAccounts(k.namespace).Get(name, v1.GetOptions{IncludeUninitialized: true})
	if k8serrors.IsNotFound(err) {
		return nil, errors.NotFoundf("service account %q", name)
	}
	return out, errors.Trace(err)
}

func (k *kubernetesClient) deleteServiceAccount(name string) error {
	err := k.client().CoreV1().ServiceAccounts(k.namespace).Delete(name, &v1.DeleteOptions{
		PropagationPolicy: &defaultPropagationPolicy,
	})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return errors.Trace(err)
}

func (k *kubernetesClient) deleteServiceAccounts(appName string) error {
	err := k.client().CoreV1().ServiceAccounts(k.namespace).DeleteCollection(&v1.DeleteOptions{
		PropagationPolicy: &defaultPropagationPolicy,
	}, v1.ListOptions{
		LabelSelector:        labelsToSelector(k.getRBACLabels(appName)),
		IncludeUninitialized: true,
	})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return errors.Trace(err)
}

func (k *kubernetesClient) listServiceAccount(appName string) ([]core.ServiceAccount, error) {
	listOps := v1.ListOptions{
		LabelSelector:        applicationSelector(appName),
		IncludeUninitialized: true,
	}
	saList, err := k.client().CoreV1().ServiceAccounts(k.namespace).List(listOps)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(saList.Items) == 0 {
		return nil, errors.NotFoundf("service account for application %q", appName)
	}
	return saList.Items, nil
}

func (k *kubernetesClient) deleteServiceAccountsRolesBindings(appName string) error {
	if err := k.deleteRoleBindings(appName); err != nil {
		return errors.Trace(err)
	}
	if err := k.deleteClusterRoleBindings(appName); err != nil {
		return errors.Trace(err)
	}
	if err := k.deleteRoles(appName); err != nil {
		return errors.Trace(err)
	}
	if err := k.deleteClusterRoles(appName); err != nil {
		return errors.Trace(err)
	}
	if err := k.deleteServiceAccounts(appName); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (k *kubernetesClient) createRole(role *rbacv1.Role) (*rbacv1.Role, error) {
	out, err := k.client().RbacV1().Roles(k.namespace).Create(role)
	if k8serrors.IsAlreadyExists(err) {
		return nil, errors.AlreadyExistsf("role %q", role.GetName())
	}
	return out, errors.Trace(err)
}

func (k *kubernetesClient) updateRole(role *rbacv1.Role) (*rbacv1.Role, error) {
	out, err := k.client().RbacV1().Roles(k.namespace).Update(role)
	if k8serrors.IsNotFound(err) {
		return nil, errors.NotFoundf("role %q", role.GetName())
	}
	return out, errors.Trace(err)
}

func (k *kubernetesClient) ensureRole(role *rbacv1.Role) (*rbacv1.Role, error) {
	out, err := k.updateRole(role)
	if errors.IsNotFound(err) {
		out, err = k.createRole(role)
	}
	return out, errors.Trace(err)
}

func (k *kubernetesClient) getRole(name string) (*rbacv1.Role, error) {
	out, err := k.client().RbacV1().Roles(k.namespace).Get(name, v1.GetOptions{IncludeUninitialized: true})
	if k8serrors.IsNotFound(err) {
		return nil, errors.NotFoundf("role %q", name)
	}
	return out, errors.Trace(err)
}

func (k *kubernetesClient) deleteRole(name string) error {
	err := k.client().RbacV1().Roles(k.namespace).Delete(name, &v1.DeleteOptions{
		PropagationPolicy: &defaultPropagationPolicy,
	})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return errors.Trace(err)
}

func (k *kubernetesClient) deleteRoles(appName string) error {
	err := k.client().RbacV1().Roles(k.namespace).DeleteCollection(&v1.DeleteOptions{
		PropagationPolicy: &defaultPropagationPolicy,
	}, v1.ListOptions{
		LabelSelector:        labelsToSelector(k.getRBACLabels(appName)),
		IncludeUninitialized: true,
	})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return errors.Trace(err)
}

func (k *kubernetesClient) createClusterRole(cRole *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	out, err := k.client().RbacV1().ClusterRoles().Create(cRole)
	if k8serrors.IsAlreadyExists(err) {
		return nil, errors.AlreadyExistsf("cluster role %q", cRole.GetName())
	}
	return out, errors.Trace(err)
}

func (k *kubernetesClient) updateClusterRole(cRole *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	out, err := k.client().RbacV1().ClusterRoles().Update(cRole)
	if k8serrors.IsNotFound(err) {
		return nil, errors.NotFoundf("cluster role %q", cRole.GetName())
	}
	return out, errors.Trace(err)
}

func (k *kubernetesClient) ensureClusterRole(cRole *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	out, err := k.updateClusterRole(cRole)
	if errors.IsNotFound(err) {
		out, err = k.createClusterRole(cRole)
	}
	return out, errors.Trace(err)
}

func (k *kubernetesClient) getClusterRole(name string) (*rbacv1.ClusterRole, error) {
	out, err := k.client().RbacV1().ClusterRoles().Get(name, v1.GetOptions{IncludeUninitialized: true})
	if k8serrors.IsNotFound(err) {
		return nil, errors.NotFoundf("cluster role %q", name)
	}
	return out, errors.Trace(err)
}

func (k *kubernetesClient) deleteClusterRole(name string) error {
	err := k.client().RbacV1().ClusterRoles().Delete(name, &v1.DeleteOptions{
		PropagationPolicy: &defaultPropagationPolicy,
	})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return errors.Trace(err)
}

func (k *kubernetesClient) deleteClusterRoles(appName string) error {
	err := k.client().RbacV1().ClusterRoles().DeleteCollection(&v1.DeleteOptions{
		PropagationPolicy: &defaultPropagationPolicy,
	}, v1.ListOptions{
		LabelSelector:        labelsToSelector(k.getRBACLabels(appName)),
		IncludeUninitialized: true,
	})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return errors.Trace(err)
}

func (k *kubernetesClient) createRoleBinding(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	out, err := k.client().RbacV1().RoleBindings(k.namespace).Create(rb)
	if k8serrors.IsAlreadyExists(err) {
		return nil, errors.AlreadyExistsf("role binding %q", rb.GetName())
	}
	return out, errors.Trace(err)
}

func (k *kubernetesClient) updateRoleBinding(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	out, err := k.client().RbacV1().RoleBindings(k.namespace).Update(rb)
	if k8serrors.IsNotFound(err) {
		return nil, errors.NotFoundf("role binding %q", rb.GetName())
	}
	return out, errors.Trace(err)
}

func (k *kubernetesClient) ensureRoleBinding(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	out, err := k.updateRoleBinding(rb)
	if errors.IsNotFound(err) {
		out, err = k.createRoleBinding(rb)
	}
	return out, errors.Trace(err)
}

func (k *kubernetesClient) getRoleBinding(name string) (*rbacv1.RoleBinding, error) {
	out, err := k.client().RbacV1().RoleBindings(k.namespace).Get(name, v1.GetOptions{IncludeUninitialized: true})
	if k8serrors.IsNotFound(err) {
		return nil, errors.NotFoundf("role binding %q", name)
	}
	return out, errors.Trace(err)
}

func (k *kubernetesClient) deleteRoleBinding(name string) error {
	err := k.client().RbacV1().RoleBindings(k.namespace).Delete(name, &v1.DeleteOptions{
		PropagationPolicy: &defaultPropagationPolicy,
	})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return errors.Trace(err)
}

func (k *kubernetesClient) deleteRoleBindings(appName string) error {
	err := k.client().RbacV1().RoleBindings(k.namespace).DeleteCollection(&v1.DeleteOptions{
		PropagationPolicy: &defaultPropagationPolicy,
	}, v1.ListOptions{
		LabelSelector:        labelsToSelector(k.getRBACLabels(appName)),
		IncludeUninitialized: true,
	})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return errors.Trace(err)
}

func (k *kubernetesClient) createClusterRoleBinding(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	out, err := k.client().RbacV1().ClusterRoleBindings().Create(crb)
	if k8serrors.IsAlreadyExists(err) {
		return nil, errors.AlreadyExistsf("cluster role binding %q", crb.GetName())
	}
	return out, errors.Trace(err)
}

func (k *kubernetesClient) updateClusterRoleBinding(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	out, err := k.client().RbacV1().ClusterRoleBindings().Update(crb)
	if k8serrors.IsNotFound(err) {
		return nil, errors.NotFoundf("cluster role binding %q", crb.GetName())
	}
	return out, errors.Trace(err)
}

func (k *kubernetesClient) ensureClusterRoleBinding(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	out, err := k.updateClusterRoleBinding(crb)
	if errors.IsNotFound(err) {
		out, err = k.createClusterRoleBinding(crb)
	}
	return out, errors.Trace(err)
}

func (k *kubernetesClient) getClusterRoleBinding(name string) (*rbacv1.ClusterRoleBinding, error) {
	out, err := k.client().RbacV1().ClusterRoleBindings().Get(name, v1.GetOptions{IncludeUninitialized: true})
	if k8serrors.IsNotFound(err) {
		return nil, errors.NotFoundf("cluster role binding %q", name)
	}
	return out, errors.Trace(err)
}

func (k *kubernetesClient) deleteClusterRoleBinding(name string) error {
	err := k.client().RbacV1().ClusterRoleBindings().Delete(name, &v1.DeleteOptions{
		PropagationPolicy: &defaultPropagationPolicy,
	})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return errors.Trace(err)
}

func (k *kubernetesClient) deleteClusterRoleBindings(appName string) error {
	err := k.client().RbacV1().ClusterRoleBindings().DeleteCollection(&v1.DeleteOptions{
		PropagationPolicy: &defaultPropagationPolicy,
	}, v1.ListOptions{
		LabelSelector:        labelsToSelector(k.getRBACLabels(appName)),
		IncludeUninitialized: true,
	})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return errors.Trace(err)
}
