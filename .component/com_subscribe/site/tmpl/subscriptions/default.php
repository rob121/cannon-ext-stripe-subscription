<?php
/**
 * @version    CVS: 1.0.0
 * @package    Com_Subscription
 * @author     Rob <pr@floridarangers.com>
 * @copyright  2024 Rob
 * @license    GNU General Public License version 2 or later; see LICENSE.txt
 */
// No direct access
defined('_JEXEC') or die;

use \Joomla\CMS\HTML\HTMLHelper;
use \Joomla\CMS\Factory;
use \Joomla\CMS\Uri\Uri;
use \Joomla\CMS\Router\Route;
use \Joomla\CMS\Language\Text;
use \Joomla\CMS\Layout\LayoutHelper;
use \Joomla\CMS\Session\Session;
use \Joomla\CMS\User\UserFactoryInterface;

HTMLHelper::_('bootstrap.tooltip');
HTMLHelper::_('behavior.multiselect');
HTMLHelper::_('formbehavior.chosen', 'select');

$user       = Factory::getApplication()->getIdentity();
$userId     = $user->get('id');
$listOrder  = $this->state->get('list.ordering');
$listDirn   = $this->state->get('list.direction');
$canCreate  = $user->authorise('core.create', 'com_subscription') && file_exists(JPATH_COMPONENT . DIRECTORY_SEPARATOR . 'forms' . DIRECTORY_SEPARATOR . 'subscriberform.xml');
$canEdit    = $user->authorise('core.edit', 'com_subscription') && file_exists(JPATH_COMPONENT .  DIRECTORY_SEPARATOR . 'forms' . DIRECTORY_SEPARATOR . 'subscriberform.xml');
$canCheckin = $user->authorise('core.manage', 'com_subscription');
$canChange  = $user->authorise('core.edit.state', 'com_subscription');
$canDelete  = $user->authorise('core.delete', 'com_subscription');

// Import CSS
$wa = $this->document->getWebAssetManager();
$wa->useStyle('com_subscription.list');
?>
<h3 class="componentheading">My Subscriptions</h3>
<div class="alert alert-primary d-flex align-items-center" role="alert">
  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="currentColor" class="bi bi-exclamation-triangle-fill flex-shrink-0 me-2" viewBox="0 0 16 16" role="img" aria-label="Warning:">
	<path d="M8.982 1.566a1.13 1.13 0 0 0-1.96 0L.165 13.233c-.457.778.091 1.767.98 1.767h13.713c.889 0 1.438-.99.98-1.767L8.982 1.566zM8 5c.535 0 .954.462.9.995l-.35 3.507a.552.552 0 0 1-1.1 0L7.1 5.995A.905.905 0 0 1 8 5zm.002 6a1 1 0 1 1 0 2 1 1 0 0 1 0-2z"/>
  </svg>
  <div>
	Select a subscription if you need to cancel it, if you need to update payment methods, cancel the subscription and click on the name to create a new one with your updated payment information
</div>
</div>

<form action="<?php echo htmlspecialchars(Uri::getInstance()->toString()); ?>" method="post"
	  name="adminForm" id="adminForm">
	
	<div class="table-responsive">
		<table class="table table-striped" id="subscriberList">
			<thead>
			<tr>
				
			  	<th >
					<?php echo HTMLHelper::_('grid.sort', 'Product', 'a.product_name', $listDirn, $listOrder); ?>
				  </th>
					<th >
						<?php echo HTMLHelper::_('grid.sort', 'Status', 'a.state', $listDirn, $listOrder); ?>
					</th>

					<th class=''>
						<?php echo HTMLHelper::_('grid.sort',  'Subscriber', 'a.user_id', $listDirn, $listOrder); ?>
					</th>

					<th class=''>
						<?php echo HTMLHelper::_('grid.sort',  'Start Date', 'a.start_date', $listDirn, $listOrder); ?>
					</th>


			</tr>
			</thead>
			<tfoot>
			<tr>
				<td colspan="<?php echo isset($this->items[0]) ? count(get_object_vars($this->items[0])) : 10; ?>">
					<div class="pagination">
						<?php echo $this->pagination->getPagesLinks(); ?>
					</div>
				</td>
			</tr>
			</tfoot>
			<tbody>
			<?php foreach ($this->items as $i => $item) : ?>
				<?php $canEdit = $user->authorise('core.edit', 'com_subscription'); ?>
				<?php
				 $canChange = false;
				 if ($item->state == 1){ 
					$canChange = true;
				}?>
					
				<tr class="row<?php echo $i % 2; ?>">
					
					<td>
						<a  href="<?php echo ($canChange) ? Route::_('index.php?option=com_subscription&view=subscriber&id=' . $item->id, false, 2) : Route::_('index.php?option=com_subscription&view=product&id=' . $item->product_id, false, 2); ?>">
						<?php echo $item->products_name; ?>
						</a>
					</td>
					<td>
						<?php $class = ($canChange) ? 'btn-success' : 'btn-danger'; ?>
						<a class="btn btn-micro <?php echo $class; ?>" href="<?php echo ($canChange) ? Route::_('index.php?option=com_subscription&view=subscriber&id=' . $item->id, false, 2) : Route::_('index.php?option=com_subscription&view=product&id=' . $item->product_id, false, 2); ?>">
						<?php if ($item->state == 1): ?>
							<i class="icon-publish"> </i> Active
						<?php else: ?>
							<i class="icon-unpublish"></i> Cancelled
						<?php endif; ?>
						</a>
					</td>
					<td>
								<?php $container = \Joomla\CMS\Factory::getContainer();
								$userFactory = $container->get(UserFactoryInterface::class);?>
								<?php $user = $userFactory->loadUserById($item->user_id); ?>
								<?php echo $user->name; ?>
					</td>
					<td>
						<?php
						$date = $item->start_date;
						echo $date > 0 ? HTMLHelper::_('date', $date, Text::_('DATE_FORMAT_LC4')) : '-';
						?>
					</td>
				

				</tr>
			<?php endforeach; ?>
			</tbody>
		</table>
	</div>
	<?php if ($canCreate) : ?>
		<a href="<?php echo Route::_('index.php?option=com_subscription&task=subscriberform.edit&id=0', false, 0); ?>"
		   class="btn btn-success btn-small"><i
				class="icon-plus"></i>
			<?php echo Text::_('COM_SUBSCRIPTION_ADD_ITEM'); ?></a>
	<?php endif; ?>

	<input type="hidden" name="task" value=""/>
	<input type="hidden" name="boxchecked" value="0"/>
	<input type="hidden" name="filter_order" value=""/>
	<input type="hidden" name="filter_order_Dir" value=""/>
	<?php echo HTMLHelper::_('form.token'); ?>
</form>

<?php
	if($canDelete) {
		$wa->addInlineScript("
			jQuery(document).ready(function () {
				jQuery('.delete-button').click(deleteItem);
			});

			function deleteItem() {

				if (!confirm(\"" . Text::_('COM_SUBSCRIPTION_DELETE_MESSAGE') . "\")) {
					return false;
				}
			}
		", [], [], ["jquery"]);
	}
?>