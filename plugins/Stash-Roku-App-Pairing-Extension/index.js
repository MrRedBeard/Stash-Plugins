// index.js (embedded JS plugin)
PluginApi.register.route('/plugin/roku-pair', () =>
{
    const { React, components } = PluginApi;
    const { useEffect, useState } = React;

    function ConfirmPage()
    {
        const params = new URLSearchParams(window.location.search);
        const rid = params.get('rid');
        const [state, setState] = useState({ done: false, err: null });

        async function confirm()
        {
            try
            {
                const res = await fetch('http://' + window.location.host.replace(/:.*/, '') + ':9998/roku/pair/confirm', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ rid })
                });
                if (!res.ok) throw new Error('Confirm failed');
                setState({ done: true, err: null });
            } catch (e) { setState({ done: false, err: e.message }); }
        }

        return (
            React.createElement('div', { style: { padding: 20, maxWidth: 480 } },
                React.createElement('h2', null, 'Pair Roku'),
                React.createElement('p', null, 'Confirm pairing for this Roku device?'),
                React.createElement('button', { onClick: confirm }, 'Confirm'),
                state.done && React.createElement('p', null, 'Paired! You can return to your TV.'),
                state.err && React.createElement('p', { style: { color: 'red' } }, state.err)
            )
        );
    }

    return ConfirmPage;
});
