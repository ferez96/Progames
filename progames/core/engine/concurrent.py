__all__ = ["run", "DEFAULT_POOL"]

from multiprocessing.pool import ThreadPool

POOL_SIZE = 20
POOL_INITIALIZER = None
POOL_INIT_ARGS = ()
DEFAULT_POOL = ThreadPool(
    processes=POOL_SIZE,
    initializer=POOL_INITIALIZER,
    *POOL_INIT_ARGS,
)


def run(func, args=None, kwargs=None):
    """execute a function in default thread pool

    Args:
        func (function): function to be executed
        args (tuple): args
        kwargs (dict): kwargs

    Returns:
        None
    """
    if args is None:
        args = ()
    if kwargs is None:
        kwargs = {}
    return DEFAULT_POOL.apply(func, *args, **kwargs)
