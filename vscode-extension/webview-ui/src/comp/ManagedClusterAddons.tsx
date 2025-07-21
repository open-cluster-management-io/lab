import { VSCodeDataGrid, VSCodeDataGridCell, VSCodeDataGridRow,  } from '@vscode/webview-ui-toolkit/react';
import { useState, useEffect } from 'react';
import { OcmResource } from '../../../src/data/loader'

export default function ShowManagedClusterAddons() {
    let [managedClusterAddons, setManagedClusterAddons] = useState<OcmResource[]>([]);

	useEffect(() => {
        window.addEventListener("message", event => {
			if ('crsDistribution' in event.data && 'ManagedClusterAddOn' === event.data.crsDistribution.kind) {
				setManagedClusterAddons(JSON.parse(event.data.crsDistribution.crs));
			}
        });
    });

    return (
        <section className="component-row">
            { managedClusterAddons.length > 0 &&
                <>
                    <h2 style={{ marginTop: '40px' }}>Managed Cluster Addons</h2>
                    <VSCodeDataGrid gridTemplateColumns="1fr 1fr 2fr" aria-label='ManagedClusterAddons' >
                        <VSCodeDataGridRow rowType="sticky-header">
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='1'>Managed Cluster Addon Name</VSCodeDataGridCell>
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='2'>Namespace</VSCodeDataGridCell>
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='3'>Conditions</VSCodeDataGridCell>
                        </VSCodeDataGridRow>

                        {managedClusterAddons.map(managedClusterAddon => {
                            return <VSCodeDataGridRow>
                                        <VSCodeDataGridCell gridColumn='1'>{managedClusterAddon.kr.metadata.name}</VSCodeDataGridCell>
                                        <VSCodeDataGridCell gridColumn='2'>{managedClusterAddon.kr.metadata.namespace} </VSCodeDataGridCell>
                                        <VSCodeDataGridCell gridColumn='3'>{managedClusterAddon.kr.status.conditions.map( ( condition:any )=> { return<p> - lastTransitionTime: {condition.lastTransitionTime}, message: {condition.message}, reason: {condition.reason}, status: {condition.status}, type: {condition.type} </p>  })} </VSCodeDataGridCell>
									</VSCodeDataGridRow>
                        } )
                        }
                    </VSCodeDataGrid>
                    <div style={{ borderTop: "1px solid #fff ", marginLeft: 10, marginRight: 10 }}></div>
                </>
            }
        </section>
    );
}
