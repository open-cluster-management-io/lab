import React from "react";
import ShowSelectedContext from "./comp/SelectedContext";
import ShowManagedClusters from "./comp/ManagedClusters";
import ShowManifestWorks from "./comp/ManifestWorks";
import ShowPlacements from "./comp/Placements";
import ShowPlacementDecisions from "./comp/PlacementDecisions";
import ShowManagedClusterSets from "./comp/ManagedClusterSets";
import ShowManagedClusterAddons from "./comp/ManagedClusterAddons";
import ShowClusterManagers from "./comp/ClusterManagers";
import ShowSubscriptionReports from "./comp/SubscriptionReports";
import ShowKlusterlets from "./comp/Klusterlets";
import ShowAppliedManifestWorks from "./comp/AppliedManifestWorks";
import ShowSubscriptionStatuses from "./comp/SubscriptionStatuses";
export default class App extends React.Component {
	render(): JSX.Element {
		return (
			<main>
				<ShowSelectedContext/>

				<ShowClusterManagers/>

				<ShowManagedClusters/>

				<ShowManifestWorks/>

				<ShowSubscriptionReports/>

				<ShowPlacements/>

				<ShowPlacementDecisions/>

				<ShowManagedClusterSets/>

				<ShowManagedClusterAddons/>

				<ShowKlusterlets/>

				<ShowAppliedManifestWorks/>

				<ShowSubscriptionStatuses/>

			</main>
		)
	}
}
